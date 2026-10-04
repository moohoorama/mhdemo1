# coding: utf-8
"""Play the built CLI through JSON Lines, including saves at scenario/AI boundaries."""
import json,pathlib,subprocess,tempfile
root=pathlib.Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='srpg-smoke-') as directory:
    proc=subprocess.Popen([str(root/'bin/srpg-cli'),'-json','-data',directory],cwd=root,stdin=subprocess.PIPE,stdout=subprocess.PIPE,text=True)
    count=0
    def call(op,**args):
        global count
        count+=1
        proc.stdin.write(json.dumps(dict(id=count,op=op,**args),ensure_ascii=False)+'\n');proc.stdin.flush()
        result=json.loads(proc.stdout.readline())
        assert result['ok'],result
        return result.get('observation')
    state=call('new',seed=1)
    call('settings',settings=dict(ai='step',color=False,detail=False,text_delay_ms=0))
    rounds=[];saved=0
    for step in range(5000):
        if step%47==0:
            call('save',slot='1',overwrite=True)
            restored=call('load',slot='1')
            assert restored==state
            saved+=1
        phase=state['Phase'];rev=state['Revision']
        if phase=='complete':
            assert all(state['Executed'].get(key) for key in ['join-간옹','grant-고정도','reward-B01','reward-B02','reward-B03'])
            break
        if phase=='scenario':
            kind=state['Dialogue']['Kind']
            if kind=='dialogue':command=dict(kind='next')
            elif kind=='choice':
                call('save',slot='2',overwrite=True);assert call('load',slot='2')==state
                command=dict(kind='choose',option='결의')
            elif state['Warehouse'].get('item_024',0):command=dict(kind='equip',actor='유비',item='item_024')
            else:command=dict(kind='start')
            state=call('command',revision=rev,command=command)
        elif phase=='battle':state=call('auto' if state['Turn']=='ally' else 'ai',revision=rev)
        elif phase=='result':
            assert state['Result']=='victory',state['Stage']
            rounds.append(state['Round']);state=call('command',revision=rev,command=dict(kind='continue'))
    else:raise AssertionError('campaign did not finish')
    call('save',slot='10',overwrite=True)
    assert call('load',slot='10')==state
    call('quit');assert proc.wait(timeout=5)==0
    print(json.dumps(dict(result='complete',requests=count,restores=saved+2,rounds=rounds),ensure_ascii=False))
