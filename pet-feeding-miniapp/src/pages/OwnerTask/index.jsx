import {useState} from 'react'
import {Navigate, useNavigate, useParams} from 'react-router-dom'
import {Clock, Flame, MapPin, MessageCircle, Timer, TrendingDown} from 'lucide-react'
import NavBar from '@/components/NavBar'
import {Rating, Row, Section, Sheet, StatusTag, Verified} from '@/components/Common'
import {STUDENTS} from '@/data/mock'
import {petOf, useApp} from '@/store/AppContext'
import {fmtTime, useCountdown} from '@/utils/useCountdown'

function Summary({task}) {
    const pet = petOf(task)
    return (
        <div className="card flex gap-3">
            <img src={pet.img} alt={`${pet.name} 照片`} className="h-16 w-16 rounded-2xl object-cover" />
            <div className="flex-1">
                <div className="flex items-center justify-between">
                    <b>{pet.name} · {pet.desc}</b>
                    <StatusTag status={task.status} />
                </div>
                <p className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                    <Clock size={12} /> {task.day} {task.window}
                </p>
                <p className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground">
                    <MapPin size={12} /> {task.community}
                </p>
                <p className="mt-1 text-xs">{task.services.join(' · ')}　<b className="text-primary-600">预算 ≤ ¥{task.budget}</b></p>
            </div>
        </div>
    )
}

function BidCard({bid, lowest, onPick}) {
    const s = STUDENTS[bid.studentId]
    return (
        <div className="card">
            <div className="flex items-center gap-3">
                <img src={s.avatar} alt={`${s.name} 头像`} className="h-11 w-11 rounded-full object-cover" />
                <div className="flex-1">
                    <div className="flex items-center gap-1.5">
                        <b className="text-sm">{s.name}{bid.studentId === 's0' && '（我）'}</b>
                        <Verified />
                    </div>
                    <Rating value={s.rating} done={s.done} />
                    <span className="ml-2 text-[11px] text-muted-foreground">距你约 {s.dist}km</span>
                </div>
                <div className="text-right">
                    <b className="text-xl text-primary-600">¥{bid.price}</b>
                    {lowest && (
                        <p className="flex items-center justify-end gap-0.5 text-[10px] text-secondary-600">
                            <TrendingDown size={11} />最低价
                        </p>
                    )}
                </div>
            </div>
            {bid.msg && (
                <p className="mt-2.5 flex items-start gap-1.5 rounded-xl bg-muted p-2.5 text-xs text-muted-foreground">
                    <MessageCircle size={13} className="mt-0.5 shrink-0" />
                    {bid.msg}
                </p>
            )}
            <button type="button" onClick={() => onPick(bid)} className="btn-primary mt-3 w-full py-2.5">选 TA 上门</button>
        </div>
    )
}

function PayPanel({task, onPay, onReopen}) {
    const left = useCountdown(task.payRemain || 600)
    const bid = task.bids.find((b) => b.id === task.winnerBidId)
    const s = STUDENTS[bid.studentId]
    return (
        <Section title="确认并支付">
            <div className="card">
                <div className="flex items-center gap-3">
                    <img src={s.avatar} alt={`${s.name} 头像`} className="h-11 w-11 rounded-full object-cover" />
                    <div className="flex-1"><b>{s.name}</b> <Verified /></div>
                    <b className="text-xl text-primary-600">¥{task.price}</b>
                </div>
                <div className="mt-3 flex items-center justify-between rounded-xl bg-accent-50 p-3 text-xs text-accent-800">
                    <span className="flex items-center gap-1"><Timer size={14} />中选资格保留，请尽快支付</span>
                    <b className="animate-pulse-soft font-mono text-base">{fmtTime(left)}</b>
                </div>
                <p className="mt-2 text-[11px] text-muted-foreground">支付总价即中选报价，不会在支付页增加额外费用。</p>
                <button type="button" onClick={onPay} className="btn-primary mt-3 w-full">微信支付 ¥{task.price}</button>
                <button type="button" onClick={onReopen} className="btn-ghost mt-2 w-full">换人选择 / 重新竞价</button>
            </div>
        </Section>
    )
}

export default function OwnerTask() {
    const {id} = useParams()
    const navigate = useNavigate()
    const {tasks, selectBid, reopen, pay, cancel, toast} = useApp()
    const [pick, setPick] = useState(null)
    const task = tasks.find((t) => t.id === id)
    const left = useCountdown(task?.remain || 0)

    if (!task) {
        return <Navigate to="/owner" replace />
    }
    if (!['BIDDING', 'PENDING_PAYMENT'].includes(task.status)) {
        return <Navigate to={`/owner/order/${id}`} replace />
    }
    const bids = [...task.bids].sort((a, b) => a.price - b.price)

    return (
        <div>
            <NavBar title={task.status === 'BIDDING' ? '报价详情' : '待付款'} backTo="/owner" />
            <div className="space-y-4 px-4 pt-2">
                <Summary task={task} />
                {task.status === 'PENDING_PAYMENT' && (
                    <PayPanel
                        task={task}
                        onPay={() => {
                            pay(task.id)
                            toast('支付成功，订单已预约')
                            navigate(`/owner/order/${task.id}`, {replace: true})
                        }}
                        onReopen={() => {
                            reopen(task.id)
                            toast('已重新开放竞价')
                        }}
                    />
                )}
                {task.status === 'BIDDING' && (
                    <>
                        <div className="flex items-center justify-between rounded-2xl bg-gradient-to-r from-primary-500 to-accent-500 px-4 py-3 text-white">
                            <span className="flex items-center gap-1.5 text-sm"><Flame size={16} />竞价进行中</span>
                            <span className="text-xs">剩余 <b className="font-mono text-base">{fmtTime(left)}</b></span>
                        </div>
                        <Section title={`已收到 ${bids.length} 个报价`} className="mt-0" extra={<span className="text-[11px] text-muted-foreground">不会自动选最低价</span>}>
                            <div className="space-y-3">
                                {bids.map((b, i) => <BidCard key={b.id} bid={b} lowest={i === 0 && bids.length > 1} onPick={setPick} />)}
                                {!bids.length && (
                                    <div className="card py-8 text-center text-sm text-muted-foreground">
                                        暂无报价，附近学生已收到通知
                                        <button type="button" onClick={() => navigate(`/student/task/${task.id}`)} className="btn-secondary mx-auto mt-4 block">
                                            去学生端体验报价
                                        </button>
                                    </div>
                                )}
                            </div>
                        </Section>
                        <button
                            type="button"
                            onClick={() => {
                                cancel(task.id)
                                toast('任务已撤单，报价自动失效')
                                navigate('/owner', {replace: true})
                            }}
                            className="w-full py-2 text-center text-xs text-muted-foreground underline"
                        >
                            撤销任务（未付款无需退款）
                        </button>
                    </>
                )}
            </div>

            <Sheet open={!!pick} onClose={() => setPick(null)} title="确认选择">
                {pick && (
                    <>
                        <Row label="服务者">{STUDENTS[pick.studentId].name}（已核验）</Row>
                        <Row label="成交价">¥{pick.price}</Row>
                        <Row label="服务时间">{task.day} {task.window}</Row>
                        <p className="mt-2 rounded-xl bg-accent-50 p-3 text-xs text-accent-800">
                            选定后任务将锁定，请在 10 分钟内完成支付；其他报价暂时冻结。
                        </p>
                        <button
                            type="button"
                            onClick={() => {
                                selectBid(task.id, pick.id)
                                setPick(null)
                                toast('已锁定，请完成支付')
                            }}
                            className="btn-primary mt-4 w-full"
                        >
                            确认选择并去支付
                        </button>
                    </>
                )}
            </Sheet>
        </div>
    )
}
