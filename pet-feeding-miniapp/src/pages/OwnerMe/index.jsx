import {useNavigate} from 'react-router-dom'
import {Bell, ChevronRight, FileText, Headphones, Heart, MapPin, PawPrint, Repeat, ShieldCheck} from 'lucide-react'
import NavBar from '@/components/NavBar'
import {AVATAR} from '@/data/mock'
import {useApp} from '@/store/AppContext'

const MENU = [
    [PawPrint, '宠物档案'],
    [MapPin, '地址管理'],
    [Bell, '订阅消息授权'],
    [ShieldCheck, '隐私设置'],
    [FileText, '服务协议与退款规则'],
    [Headphones, '联系客服'],
]

export default function OwnerMe() {
    const navigate = useNavigate()
    const {toast, tasks} = useApp()
    const done = tasks.filter((t) => t.ownerId === 'o0' && t.status === 'COMPLETED').length
    return (
        <div>
            <NavBar back={false} title="我的" />
            <div className="space-y-4 px-4 pt-3">
                <div className="card flex items-center gap-4 bg-gradient-to-br from-primary-50 to-white">
                    <img src={AVATAR.s3} alt="用户头像" className="h-16 w-16 rounded-full border-2 border-white object-cover shadow" />
                    <div className="flex-1">
                        <b className="text-lg">阿宁</b>
                        <p className="text-xs text-muted-foreground">138****6021 · 已实名</p>
                    </div>
                    <div className="text-center">
                        <b className="text-xl text-primary-600">{done}</b>
                        <p className="text-[11px] text-muted-foreground">完成订单</p>
                    </div>
                </div>

                <button
                    type="button"
                    onClick={() => navigate('/student')}
                    className="flex w-full items-center gap-3 rounded-2xl bg-gradient-to-r from-secondary-500 to-secondary-700 p-4 text-left text-white shadow-lg shadow-secondary-500/20 active:scale-[0.98]"
                >
                    <Repeat size={22} />
                    <div className="flex-1">
                        <b>切换为学生接单身份</b>
                        <p className="text-xs text-white/80">已通过认证的学生可在空闲时间接单</p>
                    </div>
                    <ChevronRight size={18} />
                </button>

                <div className="card divide-y divide-border/60 p-0">
                    {MENU.map(([Icon, label]) => (
                        <button key={label} type="button" onClick={() => toast(`${label}（原型示意）`)} className="flex w-full items-center gap-3 p-3.5 text-left">
                            <Icon size={18} className="text-primary-500" />
                            <span className="flex-1 text-sm">{label}</span>
                            <ChevronRight size={16} className="text-muted-foreground" />
                        </button>
                    ))}
                </div>
                <p className="flex items-center justify-center gap-1 text-[11px] text-muted-foreground">
                    <Heart size={12} /> 平台不提供医疗服务，仅限非医疗类室内照护
                </p>
            </div>
        </div>
    )
}
