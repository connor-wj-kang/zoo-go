import {useState} from 'react'
import {Navigate, useNavigate, useParams} from 'react-router-dom'
import {AlertTriangle, Check, MapPin, Phone, Star} from 'lucide-react'
import NavBar from '@/components/NavBar'
import {Row, Section, Sheet, StatusTag, Verified} from '@/components/Common'
import {FEE_RATE, STUDENTS} from '@/data/mock'
import {petOf, useApp} from '@/store/AppContext'
import {cn} from '@/utils/cn'

const STEPS = [['BOOKED', '已预约'], ['IN_SERVICE', '服务中'], ['PENDING_CONFIRM', '待验收'], ['COMPLETED', '已完成']]

function Timeline({status}) {
    const idx = Math.max(0, STEPS.findIndex(([k]) => k === status))
    return (
        <div className="card flex items-center">
            {STEPS.map(([k, label], i) => (
                <div key={k} className="flex flex-1 items-center">
                    <div className="flex flex-col items-center gap-1">
                        <span className={cn(
                            'flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold',
                            i <= idx ? 'bg-primary-500 text-white' : 'bg-gray-100 text-gray-400',
                        )}>
                            {i < idx ? <Check size={14} /> : i + 1}
                        </span>
                        <span className={cn('text-[11px]', i <= idx ? 'font-semibold text-foreground' : 'text-muted-foreground')}>{label}</span>
                    </div>
                    {i < STEPS.length - 1 && <span className={cn('mb-4 h-0.5 flex-1', i < idx ? 'bg-primary-400' : 'bg-gray-200')} />}
                </div>
            ))}
        </div>
    )
}

export default function OwnerOrder() {
    const {id} = useParams()
    const navigate = useNavigate()
    const {tasks, confirm, rate, cancel, toast} = useApp()
    const [sheet, setSheet] = useState('')
    const [stars, setStars] = useState(5)
    const task = tasks.find((t) => t.id === id)

    if (!task) {
        return <Navigate to="/owner/orders" replace />
    }
    if (['BIDDING', 'PENDING_PAYMENT'].includes(task.status)) {
        return <Navigate to={`/owner/task/${id}`} replace />
    }
    const pet = petOf(task)
    const bid = task.bids.find((b) => b.id === task.winnerBidId)
    const stu = bid && STUDENTS[bid.studentId]
    const fee = task.price ? (task.price * FEE_RATE).toFixed(1) : 0

    return (
        <div>
            <NavBar title="订单详情" backTo="/owner/orders" />
            <div className="space-y-4 px-4 pt-2 pb-28">
                <div className="flex items-center justify-between px-1">
                    <b className="text-lg">{pet.name} 的上门喂养</b>
                    <StatusTag status={task.status} />
                </div>
                {task.status !== 'CANCELLED' && <Timeline status={task.status} />}

                {stu && (
                    <div className="card">
                        <div className="flex items-center gap-3">
                            <img src={stu.avatar} alt={`${stu.name} 头像`} className="h-12 w-12 rounded-full object-cover" />
                            <div className="flex-1">
                                <b>{stu.name}</b> <Verified />
                                <p className="text-xs text-muted-foreground">评分 {stu.rating} · 完成 {stu.done} 单</p>
                            </div>
                            <button type="button" onClick={() => toast('已通过平台虚拟号联系')} className="rounded-full bg-secondary-50 p-2.5 text-secondary-600" aria-label="联系">
                                <Phone size={18} />
                            </button>
                        </div>
                        <div className="mt-3 flex items-start gap-2 rounded-xl bg-muted p-3 text-xs">
                            <MapPin size={14} className="mt-0.5 shrink-0 text-primary-500" />
                            <span>3 号楼 2 单元 401（仅向中选学生开放，访问留痕）。<br />门锁方式：主人到场交接 / 自有授权方式</span>
                        </div>
                    </div>
                )}

                {task.photos.length > 0 && (
                    <Section title="服务凭证" className="mt-0">
                        <div className="card">
                            <div className="grid grid-cols-3 gap-2">
                                {task.photos.map((p, i) => (
                                    <img key={`${p}-${i}`} src={p} alt={`服务照片 ${i + 1}`} className="aspect-square rounded-xl object-cover" />
                                ))}
                            </div>
                            <p className="mt-2 text-xs text-muted-foreground">已完成：{task.services.join('、')}　✓ 到场签到 ✓ 清单</p>
                        </div>
                    </Section>
                )}

                <Section title="费用明细" className="mt-0">
                    <div className="card divide-y divide-border/60 py-1">
                        <Row label="服务时间">{task.day} {task.window}</Row>
                        <Row label="支付总价（=中选报价）"><b className="text-primary-600">¥{task.price ?? '--'}</b></Row>
                        <Row label={`平台服务费 ${FEE_RATE * 100}%`}>¥{fee}</Row>
                    </div>
                </Section>
            </div>

            <div className="fixed inset-x-0 bottom-0 z-30 flex gap-2 border-t border-border bg-white p-4 pb-6">
                {task.status === 'BOOKED' && (
                    <>
                        <button type="button" onClick={() => setSheet('cancel')} className="btn-ghost flex-1">申请取消</button>
                        <button type="button" onClick={() => navigate('/student/fulfill/' + task.id)} className="btn-secondary flex-1">去学生端体验履约</button>
                    </>
                )}
                {task.status === 'IN_SERVICE' && (
                    <button type="button" onClick={() => setSheet('issue')} className="btn-ghost flex-1"><AlertTriangle size={15} />上报异常</button>
                )}
                {task.status === 'PENDING_CONFIRM' && (
                    <>
                        <button type="button" onClick={() => setSheet('issue')} className="btn-ghost flex-1">提出异议</button>
                        <button type="button" onClick={() => { confirm(task.id); toast('已验收，感谢使用') }} className="btn-primary flex-[2]">确认完成</button>
                    </>
                )}
                {task.status === 'COMPLETED' && (
                    <button type="button" disabled={task.rated} onClick={() => setSheet('rate')} className="btn-primary flex-1">
                        {task.rated ? '已评价' : '评价服务者'}
                    </button>
                )}
                {task.status === 'CANCELLED' && <p className="w-full py-2 text-center text-sm text-muted-foreground">订单已取消</p>}
            </div>

            <Sheet open={sheet === 'cancel'} onClose={() => setSheet('')} title="申请取消订单">
                <p className="text-sm text-muted-foreground">V1 阶段取消与退款由客服人工审核，规则已事先公示，可申诉。</p>
                <button type="button" onClick={() => { cancel(task.id); setSheet(''); toast('已提交，客服将在审核后退款') }} className="btn-primary mt-4 w-full">提交取消申请</button>
            </Sheet>
            <Sheet open={sheet === 'issue'} onClose={() => setSheet('')} title="上报异常 / 提出异议">
                <p className="text-sm text-muted-foreground">宠物受伤、财物争议、安全问题将暂停结算并由客服优先介入。</p>
                <button type="button" onClick={() => { setSheet(''); toast('已通知客服，结算已暂停') }} className="btn-primary mt-4 w-full">联系客服处理</button>
            </Sheet>
            <Sheet open={sheet === 'rate'} onClose={() => setSheet('')} title="评价服务者">
                <div className="flex justify-center gap-2 py-2">
                    {[1, 2, 3, 4, 5].map((n) => (
                        <button key={n} type="button" onClick={() => setStars(n)} aria-label={`${n} 星`}>
                            <Star size={34} className={n <= stars ? 'fill-accent-400 text-accent-400' : 'text-gray-300'} />
                        </button>
                    ))}
                </div>
                <button type="button" onClick={() => { rate(task.id); setSheet(''); toast('评价成功') }} className="btn-primary mt-3 w-full">提交评价</button>
            </Sheet>
        </div>
    )
}
