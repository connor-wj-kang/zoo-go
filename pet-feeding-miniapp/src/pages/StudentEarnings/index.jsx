import {Info, Wallet} from 'lucide-react'
import NavBar from '@/components/NavBar'
import {Section} from '@/components/Common'
import {FEE_RATE, calcStudentGet} from '@/data/mock'
import {useApp} from '@/store/AppContext'

const HISTORY = [
    {id: 'x1', title: '10-30 · 球球 喂食换水', total: 40, state: '已结算'},
    {id: 'x2', title: '10-24 · 咪咪 清理猫砂', total: 45, state: '已结算'},
    {id: 'x3', title: '10-18 · 布丁 喂食', total: 36, state: '已结算'},
]

export default function StudentEarnings() {
    const {tasks, toast} = useApp()
    const pending = tasks.filter((t) => t.bids.some((b) => b.id === t.winnerBidId && b.studentId === 's0') && ['PENDING_CONFIRM', 'COMPLETED'].includes(t.status))
    const pendingSum = pending.reduce((s, t) => s + calcStudentGet(t.price), 0)
    const settled = HISTORY.reduce((s, h) => s + calcStudentGet(h.total), 0)

    return (
        <div>
            <NavBar back={false} title="收益中心" />
            <div className="px-4 pt-3">
                <div className="rounded-3xl bg-gradient-to-br from-secondary-500 to-secondary-700 p-5 text-white shadow-lg shadow-secondary-500/25">
                    <p className="flex items-center gap-1.5 text-xs text-white/80"><Wallet size={14} />累计已赚取（元）</p>
                    <b className="mt-1 block text-4xl">{(settled + 0).toFixed(1)}</b>
                    <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                        <div className="rounded-2xl bg-white/15 p-3"><p className="text-xs text-white/80">待结算</p><b className="text-lg">¥{pendingSum.toFixed(1)}</b></div>
                        <div className="rounded-2xl bg-white/15 p-3"><p className="text-xs text-white/80">本月订单</p><b className="text-lg">{HISTORY.length + pending.length} 单</b></div>
                    </div>
                </div>
                <p className="mt-3 flex items-start gap-1.5 rounded-xl bg-accent-50 p-3 text-[11px] text-accent-800">
                    <Info size={13} className="mt-0.5 shrink-0" />
                    订单争议期间不会自动结算；结算渠道与到账时间以平台正式规则为准。
                </p>

                <Section title="收益明细">
                    <div className="card divide-y divide-border/60 p-0">
                        {[...pending.map((t) => ({id: t.id, title: `${t.day.slice(0, 5)} · 待结算订单`, total: t.price, state: '待结算'})), ...HISTORY].map((h) => (
                            <div key={h.id} className="flex items-center justify-between p-3.5">
                                <div>
                                    <b className="text-[13px]">{h.title}</b>
                                    <p className="text-[11px] text-muted-foreground">成交 ¥{h.total} − 服务费 ¥{(h.total * FEE_RATE).toFixed(1)}</p>
                                </div>
                                <div className="text-right">
                                    <b className="text-secondary-600">+¥{calcStudentGet(h.total)}</b>
                                    <p className="text-[11px] text-muted-foreground">{h.state}</p>
                                </div>
                            </div>
                        ))}
                    </div>
                </Section>
                <button type="button" onClick={() => toast('对账异常已反馈（原型示意）')} className="btn-ghost mx-auto mt-5 block text-xs">对账有疑问？反馈异常</button>
            </div>
        </div>
    )
}
