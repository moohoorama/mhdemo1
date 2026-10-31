package content

import (
	"slices"
	"testing"
)

// An officer drawn per class only climbs the line it has sprites for; one drawn once (원술, 손견) and a plain officer
// take every branch, and the promoted look is then the class's own.
func TestPromotesFollowsTheSpritesAnOfficerHas(t *testing.T) {
	d, err := Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		char, class string
		want        []string
	}{
		{"liubei", "light_infantry", []string{"heavy_infantry"}},
		{"liubei", "heavy_infantry", []string{"guards"}},
		{"guanyu", "light_cavalry", []string{"heavy_cavalry"}},
		{"zhangfei", "heavy_cavalry", []string{"royal_guard"}},
		{"zhaoyun", "light_cavalry", []string{"heavy_cavalry"}},
		{"lubu", "light_cavalry", []string{"heavy_cavalry", "horse_archer"}},
		{"sunjian", "light_cavalry", []string{"heavy_cavalry", "horse_archer"}},
		{"jianyong", "archer", []string{"crossbow"}},
	} {
		if got := d.Promotes(c.char, c.class); !slices.Equal(got, c.want) {
			t.Errorf("%s in %s promotes to %v, want %v", c.char, c.class, got, c.want)
		}
	}
}
