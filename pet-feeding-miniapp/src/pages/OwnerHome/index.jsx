import {useNavigate} from 'react-router-dom'
import {Bell, ClipboardEdit, Gavel, MapPin, ShieldCheck, CreditCard, Camera, Plus, EyeOff, Snowflake} from 'lucide-react'
import NavBar from '@/components/NavBar'
import OwnerTaskCard from '@/components/OwnerTaskCard'
import {Section} from '@/components/Common'
import {PETS, IMG} from '@/data/mock'
import {useApp} from '@/store/AppContext'

const STEPS = [
    [ClipboardEdit, '发布任务', '设预算与时间'],
    [Gavel, '学生报价', '择优选人'],
    [CreditCard, '付款预约', '10分钟内支付'],
    [Camera, '凭证验收', '照片+清单'],
]

const GUARANTEES = [
    [ShieldCheck, '实名+在校双重核验', '仅展示“已核验”，不展示证件信息'],
    [EyeOff, '成交前隐藏门牌', '付款后仅向中选学生短期开放'],
    [Snowflake, '争议冻结结算', '异常上报后人工客服介入'],
]

export default function OwnerHome() {
    const navigate = useNavigate()
    const {tasks, toast} = useApp()
    const mine = tasks.filter((t) => t.ownerId === 'o0' && !['COMPLETED', 'CANCELLED', 'EXPIRED'].includes(t.status))

    return (
        <div>
            <NavBar
                back={false}
                title="喂宠到家"
                right={
                    <button type="button" onClick={() => toast('暂无新消息')} aria-label="消息">
                        <Bell size={20} />
                    </button>
                }
            />
            <div className="px-4">
                <div className="mt-2 flex items-center gap-1 text-xs text-muted-foreground">
                    <MapPin size={13} className="text-primary-500" />
                    海淀区 · 马连洼街道
                    <span className="rounded-full bg-secondary-50 px-2 py-0.5 font-medium text-secondary-700">已开通</span>
                </div>

                <div className="relative mt-3 overflow-hidden rounded-3xl bg-gradient-to-br from-primary-500 to-accent-500 p-5 text-white shadow-lg shadow-primary-500/30">
                    <img src={IMG.hero} alt="宠物 banner" className="absolute -right-6 -top-2 h-44 w-44 rounded-full object-cover opacity-90" />
                    <div className="relative max-w-[60%]">
                        <p className="text-xs text-white/80">你出价格上限，学生来报价</p>
                        <h2 className="mt-1 text-xl font-bold leading-tight">出门在外，毛孩子有人管</h2>
                        <button
                            type="button"
                            onClick={() => navigate('/owner/publish')}
                            className="mt-4 inline-flex items-center gap-1 rounded-full bg-white px-4 py-2.5 text-sm font-bold text-primary-600 shadow transition active:scale-95"
                        >
                            <Plus size={16} />
                            发布喂宠任务
                        </button>
                    </div>
                </div>

                <div className="mt-4 grid grid-cols-4 gap-2">
                    {STEPS.map(([Icon, t, d], i) => (
                        <div key={t} className="card flex flex-col items-center gap-1 p-2.5 text-center">
                            <span className="flex h-9 w-9 items-center justify-center rounded-full bg-primary-50 text-primary-600">
                                <Icon size={18} />
                            </span>
                            <b className="text-[12px]">{i + 1}. {t}</b>
                            <span className="text-[10px] leading-tight text-muted-foreground">{d}</span>
                        </div>
                    ))}
                </div>

                <Section
                    title="进行中的任务"
                    extra={
                        <button type="button" onClick={() => navigate('/owner/orders')} className="text-xs text-primary-600">
                            全部订单
                        </button>
                    }
                >
                    <div className="space-y-3">
                        {mine.length ? mine.map((t) => <OwnerTaskCard key={t.id} task={t} />) : (
                            <p className="card text-center text-sm text-muted-foreground">暂无进行中的任务</p>
                        )}
                    </div>
                </Section>

                <Section title="我的宠物">
                    <div className="no-scrollbar -mx-4 flex gap-3 overflow-x-auto px-4">
                        {PETS.map((p) => (
                            <div key={p.id} className="card flex w-56 shrink-0 gap-3 p-3">
                                <img src={p.img} alt={`${p.name} 照片`} className="h-14 w-14 rounded-2xl object-cover" />
                                <div>
                                    <b className="text-sm">{p.name}</b>
                                    <p className="text-xs text-muted-foreground">{p.breed}</p>
                                    <div className="mt-1 flex gap-1">
                                        {p.flags.map((f) => (
                                            <span key={f} className="rounded bg-accent-50 px-1.5 py-0.5 text-[10px] text-accent-700">{f}</span>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        ))}
                        <button
                            type="button"
                            onClick={() => toast('添加宠物档案（原型示意）')}
                            className="flex w-24 shrink-0 flex-col items-center justify-center gap-1 rounded-2xl border border-dashed border-primary-300 bg-primary-50/50 text-xs text-primary-600"
                        >
                            <Plus size={18} />
                            添加宠物
                        </button>
                    </div>
                </Section>

                <Section title="平台保障">
                    <div className="card divide-y divide-border/60 p-0">
                        {GUARANTEES.map(([Icon, t, d]) => (
                            <div key={t} className="flex items-center gap-3 p-3.5">
                                <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-secondary-50 text-secondary-600">
                                    <Icon size={18} />
                                </span>
                                <div>
                                    <b className="text-[13px]">{t}</b>
                                    <p className="text-[11px] text-muted-foreground">{d}</p>
                                </div>
                            </div>
                        ))}
                    </div>
                </Section>
            </div>
        </div>
    )
}
