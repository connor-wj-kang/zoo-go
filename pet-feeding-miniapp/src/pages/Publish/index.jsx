import {useState} from 'react'
import {useNavigate} from 'react-router-dom'
import {Check, Info, Minus, Plus, Send} from 'lucide-react'
import NavBar from '@/components/NavBar'
import ChipGroup from '@/components/ChipGroup'
import {COMMUNITIES, DAYS, DEADLINES, FEE_RATE, PETS, SERVICE_TYPES, TIME_WINDOWS} from '@/data/mock'
import {useApp} from '@/store/AppContext'
import {cn} from '@/utils/cn'

function Field({title, children}) {
    return (
        <div className="card mt-3">
            <p className="mb-2.5 text-[13px] font-semibold">{title}</p>
            {children}
        </div>
    )
}

export default function Publish() {
    const navigate = useNavigate()
    const {addTask, toast} = useApp()
    const [petId, setPetId] = useState('p1')
    const [services, setServices] = useState(['喂食', '换水'])
    const [day, setDay] = useState(DAYS[2])
    const [win, setWin] = useState(TIME_WINDOWS[0])
    const [mode, setMode] = useState('bid')
    const [budget, setBudget] = useState(60)
    const [deadline, setDeadline] = useState(DEADLINES[2])
    const [note, setNote] = useState('')
    const [safe, setSafe] = useState(true)
    const [agree, setAgree] = useState(false)

    const submit = () => {
        if (!services.length) {
            toast('请至少选择一项服务')
            return
        }
        if (!agree) {
            toast('请先阅读并同意服务协议')
            return
        }
        const id = addTask({
            petId, mode, budget, day, window: win, services, note: note || '请按清单完成照护并拍照反馈。',
            community: COMMUNITIES[0], remain: deadline.sec,
        })
        toast('发布成功，已通知附近学生')
        navigate(`/owner/task/${id}`, {replace: true})
    }

    return (
        <div>
            <NavBar title="发布喂宠任务" />
            <div className="px-4 pb-28">
                <Field title="选择宠物">
                    <div className="space-y-2">
                        {PETS.map((p) => (
                            <button
                                key={p.id}
                                type="button"
                                onClick={() => setPetId(p.id)}
                                className={cn(
                                    'flex w-full items-center gap-3 rounded-2xl border p-2.5 text-left transition',
                                    petId === p.id ? 'border-primary-400 bg-primary-50' : 'border-border',
                                )}
                            >
                                <img src={p.img} alt={`${p.name} 照片`} className="h-12 w-12 rounded-xl object-cover" />
                                <div className="flex-1">
                                    <b className="text-sm">{p.name}</b>
                                    <p className="text-xs text-muted-foreground">{p.breed}</p>
                                </div>
                                {petId === p.id && <Check size={18} className="text-primary-600" />}
                            </button>
                        ))}
                    </div>
                </Field>

                <Field title="服务项目（可多选）">
                    <ChipGroup options={SERVICE_TYPES} value={services} onChange={setServices} multiple />
                </Field>

                <Field title="服务时间（09:00-21:00）">
                    <ChipGroup options={DAYS} value={day} onChange={setDay} />
                    <div className="mt-2.5">
                        <ChipGroup options={TIME_WINDOWS} value={win} onChange={setWin} />
                    </div>
                </Field>

                <Field title="定价方式">
                    <div className="grid grid-cols-2 gap-2">
                        {[['bid', '反向竞价', '学生保密报价，你择优'], ['fixed', '一口价', '学生抢候选，你确认']].map(([k, t, d]) => (
                            <button
                                key={k}
                                type="button"
                                onClick={() => setMode(k)}
                                className={cn(
                                    'rounded-2xl border p-3 text-left transition',
                                    mode === k ? 'border-primary-400 bg-primary-50' : 'border-border',
                                )}
                            >
                                <b className="text-sm">{t}</b>
                                <p className="mt-0.5 text-[11px] text-muted-foreground">{d}</p>
                            </button>
                        ))}
                    </div>
                    <div className="mt-4 flex items-center justify-between">
                        <span className="text-[13px] text-muted-foreground">{mode === 'bid' ? '预算上限' : '成交价'}</span>
                        <div className="flex items-center gap-3">
                            <button type="button" onClick={() => setBudget((v) => Math.max(30, v - 5))} className="rounded-full border border-border p-1.5 active:scale-90" aria-label="减少">
                                <Minus size={16} />
                            </button>
                            <b className="w-20 text-center text-2xl text-primary-600">¥{budget}</b>
                            <button type="button" onClick={() => setBudget((v) => Math.min(200, v + 5))} className="rounded-full border border-border p-1.5 active:scale-90" aria-label="增加">
                                <Plus size={16} />
                            </button>
                        </div>
                    </div>
                    <p className="mt-2 flex items-start gap-1 text-[11px] text-muted-foreground">
                        <Info size={12} className="mt-0.5 shrink-0" />
                        可设 30-200 元。含平台服务费 {FEE_RATE * 100}%，学生预计到手 ¥{(budget * (1 - FEE_RATE)).toFixed(1)}
                    </p>
                </Field>

                {mode === 'bid' && (
                    <Field title="竞价截止时间">
                        <ChipGroup options={DEADLINES} value={deadline} onChange={setDeadline} getLabel={(o) => o.label} />
                    </Field>
                )}

                <Field title="照护清单 / 注意事项">
                    <textarea
                        value={note}
                        onChange={(e) => setNote(e.target.value)}
                        rows={3}
                        placeholder="例如：每餐 1 勺半狗粮；进门后先换水；拍 2 张照片…"
                        className="w-full resize-none rounded-xl bg-muted p-3 text-[13px] outline-none focus:ring-2 focus:ring-primary-200"
                    />
                    <label className="mt-3 flex items-center gap-2 text-xs">
                        <input type="checkbox" checked={safe} onChange={(e) => setSafe(e.target.checked)} className="accent-primary-500" />
                        已确认宠物无咬人、强应激等高风险情况
                    </label>
                </Field>

                <label className="mt-4 flex items-start gap-2 px-1 text-xs text-muted-foreground">
                    <input type="checkbox" checked={agree} onChange={(e) => setAgree(e.target.checked)} className="mt-0.5 accent-primary-500" />
                    我已阅读并同意《服务协议》《隐私政策》，门牌与联系方式成交付款后才会向中选学生开放
                </label>
            </div>

            <div className="fixed inset-x-0 bottom-0 z-30 border-t border-border bg-white p-4 pb-6">
                <button type="button" onClick={submit} className="btn-primary w-full">
                    <Send size={16} />
                    {safe ? '发布任务' : '宠物有风险，将转人工审核'}
                </button>
            </div>
        </div>
    )
}
