import {useState} from 'react'
import {Navigate, useNavigate, useParams} from 'react-router-dom'
import {EyeOff, Lock, MapPin, Minus, Plus, Timer, Zap} from 'lucide-react'
import NavBar from '@/components/NavBar'
import {Row, Section} from '@/components/Common'
import {FEE_RATE, calcStudentGet} from '@/data/mock'
import {petOf, useApp} from '@/store/AppContext'
import {fmtTime, useCountdown} from '@/utils/useCountdown'

const MIN_PRICE = 20
const MAX_REV = 2

export default function StudentTask() {
    const {id} = useParams()
    const navigate = useNavigate()
    const {tasks, submitBid, withdrawBid, applyFixed, toast} = useApp()
    const task = tasks.find((t) => t.id === id)
    const mine = task?.bids.find((b) => b.studentId === 's0')
    const [price, setPrice] = useState(mine?.price ?? Math.max(MIN_PRICE, Math.round((task?.budget ?? 50) * 0.8)))
    const [msg, setMsg] = useState(mine?.msg ?? '')
    const left = useCountdown(task?.remain ?? 0)

    if (!task) {
        return <Navigate to="/student" replace />
    }
    const pet = petOf(task)
    const won = task.winnerBidId && mine && task.winnerBidId === mine.id
    const closed = task.status !== 'BIDDING'
    const revLeft = mine ? MAX_REV - mine.rev : MAX_REV

    const doSubmit = () => {
        if (price > task.budget || price < MIN_PRICE) {
            toast(`报价需在 ¥${MIN_PRICE} - ¥${task.budget} 之间`)
            return
        }
        if (mine && revLeft <= 0) {
            toast('报价修改次数已用完')
            return
        }
        submitBid(task.id, price, msg || '准时到达，按清单完成。')
        toast(mine ? '报价已修改' : '报价已提交，等待主人选择')
    }

    return (
        <div>
            <NavBar title="任务详情" backTo="/student" />
            <div className="space-y-4 px-4 pt-2 pb-32">
                <div className="card">
                    <div className="flex gap-3">
                        <img src={pet.img} alt={`${pet.name} 照片`} className="h-20 w-20 rounded-2xl object-cover" />
                        <div className="flex-1">
                            <b className="text-base">{pet.name} · {pet.desc}</b>
                            <p className="mt-1 flex items-center gap-1 text-xs text-muted-foreground"><MapPin size={12} />{task.community}</p>
                            <p className="mt-1 text-xs"><b>{task.day}</b> {task.window}</p>
                        </div>
                    </div>
                    <div className="mt-3 flex flex-wrap gap-1.5">
                        {task.services.map((s) => <span key={s} className="rounded-full bg-primary-50 px-2.5 py-1 text-[11px] text-primary-700">{s}</span>)}
                    </div>
                    <p className="mt-3 rounded-xl bg-muted p-3 text-xs leading-relaxed text-muted-foreground">{task.note}</p>
                    <p className="mt-2 flex items-center gap-1 text-[11px] text-muted-foreground">
                        <EyeOff size={12} />门牌号、联系方式在成交付款后才会向中选者开放
                    </p>
                </div>

                {won ? (
                    <div className="card bg-gradient-to-br from-secondary-50 to-white text-center">
                        <b className="text-lg text-secondary-700">恭喜，你已中选！</b>
                        <p className="mt-1 text-xs text-muted-foreground">主人已支付，请按预约时间上门履约</p>
                        <button type="button" onClick={() => navigate(`/student/fulfill/${task.id}`)} className="btn-primary mt-3">去履约</button>
                    </div>
                ) : closed ? (
                    <div className="card py-8 text-center text-sm text-muted-foreground">该任务已被选定或结束</div>
                ) : task.mode === 'fixed' ? (
                    <Section title="一口价抢单" className="mt-0">
                        <div className="card text-center">
                            <p className="text-3xl font-bold text-primary-600">¥{task.budget}</p>
                            <p className="mt-1 text-xs text-muted-foreground">预计实收 ¥{calcStudentGet(task.budget)}（含平台费 {FEE_RATE * 100}%）</p>
                            <p className="mt-3 text-[11px] text-muted-foreground">抢单仅获得候选资格，需主人确认并付款后订单才生效。</p>
                        </div>
                    </Section>
                ) : (
                    <Section title={mine ? '我的报价' : '提交报价'} className="mt-0">
                        <div className="card">
                            <div className="flex items-center justify-between text-xs text-muted-foreground">
                                <span>主人预算上限 <b className="text-foreground">¥{task.budget}</b></span>
                                <span className="flex items-center gap-1 text-primary-600"><Timer size={13} />截止 <b className="font-mono">{fmtTime(left)}</b></span>
                            </div>
                            <div className="my-4 flex items-center justify-center gap-5">
                                <button type="button" onClick={() => setPrice((v) => Math.max(MIN_PRICE, v - 1))} className="rounded-full border border-border p-2 active:scale-90" aria-label="减少"><Minus size={18} /></button>
                                <b className="w-28 text-center text-4xl text-primary-600">¥{price}</b>
                                <button type="button" onClick={() => setPrice((v) => Math.min(task.budget, v + 1))} className="rounded-full border border-border p-2 active:scale-90" aria-label="增加"><Plus size={18} /></button>
                            </div>
                            <div className="divide-y divide-border/60 rounded-xl bg-muted px-3">
                                <Row label="成交价">¥{price}</Row>
                                <Row label={`平台服务费 ${FEE_RATE * 100}%`}>-¥{(price * FEE_RATE).toFixed(1)}</Row>
                                <Row label="预计实收"><b className="text-secondary-600">¥{calcStudentGet(price)}</b></Row>
                            </div>
                            <input
                                value={msg}
                                onChange={(e) => setMsg(e.target.value)}
                                placeholder="给主人留言（可选）：经验、到达时间…"
                                className="mt-3 w-full rounded-xl bg-muted p-3 text-[13px] outline-none focus:ring-2 focus:ring-primary-200"
                            />
                            <p className="mt-3 flex items-start gap-1 text-[11px] text-muted-foreground">
                                <Lock size={12} className="mt-0.5 shrink-0" />
                                报价保密，其他学生看不到你的金额。还可修改 {revLeft} 次；中选后需承担履约责任。
                            </p>
                        </div>
                    </Section>
                )}
            </div>

            {!closed && !won && (
                <div className="fixed inset-x-0 bottom-0 z-30 flex gap-2 border-t border-border bg-white p-4 pb-6">
                    {task.mode === 'fixed' ? (
                        task.candidate === 's0' ? (
                            <button type="button" disabled className="btn-secondary flex-1">已成为候选，等待主人确认</button>
                        ) : (
                            <button type="button" onClick={() => { applyFixed(task.id); toast('已锁定候选席位') }} className="btn-primary flex-1"><Zap size={16} />申请抢单</button>
                        )
                    ) : (
                        <>
                            {mine && <button type="button" onClick={() => { withdrawBid(task.id); toast('报价已撤回') }} className="btn-ghost flex-1">撤回报价</button>}
                            <button type="button" onClick={doSubmit} className="btn-primary flex-[2]">{mine ? '修改报价' : '提交报价'}</button>
                        </>
                    )}
                </div>
            )}
        </div>
    )
}
