package autoplay

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
)

// MLP is a small fully connected network (tanh hidden layers, linear output) with its inputs and outputs
// standardized. A few thousand weights: it trains on a laptop's CPU in seconds and runs inside the engine loop.
type MLP struct {
	Sizes     []int
	W         [][]float64 // layer l: Sizes[l+1] rows of Sizes[l]
	B         [][]float64
	InMean    []float64
	InStd     []float64
	OutMean   []float64
	OutStd    []float64
	TrainLoss float64
	ValLoss   float64
}

// NewMLP makes a network with the layer sizes given, input first.
func NewMLP(sizes []int, seed uint64) *MLP {
	rng := rand.New(rand.NewPCG(seed, 1))
	m := &MLP{Sizes: sizes}
	for l := 0; l+1 < len(sizes); l++ {
		in, out := sizes[l], sizes[l+1]
		w := make([]float64, in*out)
		for i := range w {
			w[i] = rng.NormFloat64() / math.Sqrt(float64(in))
		}
		m.W = append(m.W, w)
		m.B = append(m.B, make([]float64, out))
	}
	return m
}

// forward gives every layer's activations, the standardized input first.
func (m *MLP) forward(x []float64) [][]float64 {
	acts := [][]float64{x}
	for l := range m.W {
		in, out := m.Sizes[l], m.Sizes[l+1]
		a := make([]float64, out)
		prev := acts[l]
		for o := range out {
			s := m.B[l][o]
			row := m.W[l][o*in : (o+1)*in]
			for i, v := range prev {
				s += row[i] * v
			}
			if l+1 < len(m.W) {
				s = math.Tanh(s)
			}
			a[o] = s
		}
		acts = append(acts, a)
	}
	return acts
}

func (m *MLP) norm(x []float64) []float64 {
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = (v - m.InMean[i]) / m.InStd[i]
	}
	return out
}

// Predict runs the network on raw features.
func (m *MLP) Predict(x []float64) []float64 {
	acts := m.forward(m.norm(x))
	y := acts[len(acts)-1]
	out := make([]float64, len(y))
	for i, v := range y {
		out[i] = v*m.OutStd[i] + m.OutMean[i]
	}
	return out
}

func stats(rows [][]float64) (mean, std []float64) {
	n := len(rows[0])
	mean, std = make([]float64, n), make([]float64, n)
	for _, r := range rows {
		for i, v := range r {
			mean[i] += v / float64(len(rows))
		}
	}
	for _, r := range rows {
		for i, v := range r {
			std[i] += (v - mean[i]) * (v - mean[i]) / float64(len(rows))
		}
	}
	for i := range std {
		std[i] = max(1e-6, math.Sqrt(std[i]))
	}
	return mean, std
}

// Fit trains on the rows with Adam and mean squared error, holding back the rows of a tenth of the groups
// (whole battles: rows of one battle share its end) to report the validation loss. Both losses are
// on the standardized outputs: 1 is no better than guessing the mean.
func (m *MLP) Fit(X, Y [][]float64, groups []int, epochs, batch int, lr float64, seed uint64) {
	m.InMean, m.InStd = stats(X)
	m.OutMean, m.OutStd = stats(Y)
	rng := rand.New(rand.NewPCG(seed, 2))
	var val, train []int
	for i := range X {
		if heldOut(groups[i]) {
			val = append(val, i)
		} else {
			train = append(train, i)
		}
	}
	xs := make([][]float64, len(X))
	ys := make([][]float64, len(Y))
	for i := range X {
		xs[i] = m.norm(X[i])
		ys[i] = make([]float64, len(Y[i]))
		for j, v := range Y[i] {
			ys[i][j] = (v - m.OutMean[j]) / m.OutStd[j]
		}
	}
	mw, vw := zerosLike(m.W), zerosLike(m.W)
	mb, vb := zerosLike(m.B), zerosLike(m.B)
	gw, gb := zerosLike(m.W), zerosLike(m.B)
	step := 0
	const b1, b2, eps = .9, .999, 1e-8
	for range epochs {
		rng.Shuffle(len(train), func(i, j int) { train[i], train[j] = train[j], train[i] })
		for s := 0; s < len(train); s += batch {
			end := min(len(train), s+batch)
			clear2(gw)
			clear2(gb)
			for _, k := range train[s:end] {
				m.backward(xs[k], ys[k], gw, gb, 1/float64(end-s))
			}
			step++
			c1, c2 := 1-math.Pow(b1, float64(step)), 1-math.Pow(b2, float64(step))
			adam := func(p, g, mm, vv [][]float64) {
				for l := range p {
					for i := range p[l] {
						mm[l][i] = b1*mm[l][i] + (1-b1)*g[l][i]
						vv[l][i] = b2*vv[l][i] + (1-b2)*g[l][i]*g[l][i]
						p[l][i] -= lr * (mm[l][i] / c1) / (math.Sqrt(vv[l][i]/c2) + eps)
					}
				}
			}
			adam(m.W, gw, mw, vw)
			adam(m.B, gb, mb, vb)
		}
	}
	m.TrainLoss, m.ValLoss = m.loss(xs, ys, train), m.loss(xs, ys, val)
}

// heldOut picks a tenth of the groups, spread over battles and seeds alike.
func heldOut(group int) bool { return (uint64(group)*0x9e3779b97f4a7c15>>32)%10 == 9 }

func (m *MLP) loss(xs, ys [][]float64, idx []int) float64 {
	sum := 0.0
	for _, k := range idx {
		acts := m.forward(xs[k])
		for j, v := range acts[len(acts)-1] {
			sum += (v - ys[k][j]) * (v - ys[k][j])
		}
	}
	return sum / float64(max(1, len(idx)*len(ys[0])))
}

// backward adds the gradient of one row's squared error, scaled, to gw and gb.
func (m *MLP) backward(x, y []float64, gw, gb [][]float64, scale float64) {
	acts := m.forward(x)
	top := acts[len(acts)-1]
	delta := make([]float64, len(top))
	for i := range top {
		delta[i] = 2 * (top[i] - y[i]) * scale
	}
	m.backprop(acts, delta, gw, gb)
}

func (m *MLP) backprop(acts [][]float64, delta []float64, gw, gb [][]float64) {
	for l := len(m.W) - 1; l >= 0; l-- {
		in := m.Sizes[l]
		prev := acts[l]
		var back []float64
		if l > 0 {
			back = make([]float64, in)
		}
		for o, d := range delta {
			gb[l][o] += d
			row := m.W[l][o*in : (o+1)*in]
			grow := gw[l][o*in : (o+1)*in]
			for i, v := range prev {
				grow[i] += d * v
				if back != nil {
					back[i] += d * row[i]
				}
			}
		}
		if l > 0 {
			for i := range back {
				back[i] *= 1 - prev[i]*prev[i] // tanh'
			}
		}
		delta = back
	}
}

func zerosLike(p [][]float64) [][]float64 {
	out := make([][]float64, len(p))
	for i := range p {
		out[i] = make([]float64, len(p[i]))
	}
	return out
}

func clear2(p [][]float64) {
	for i := range p {
		clear(p[i])
	}
}

// Save writes the network as JSON.
func (m *MLP) Save(path string) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// LoadMLP reads a network Save wrote.
func LoadMLP(path string) (*MLP, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m MLP
	return &m, json.Unmarshal(b, &m)
}
