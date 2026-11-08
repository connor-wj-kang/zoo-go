import {PawPrint, GraduationCap, UserRound} from 'lucide-react'
import {cn} from '@/utils/cn'

const PAGES = {
    owner: [
        ['首页', '/owner'],
        ['发布任务', '/owner/publish'],
        ['报价详情（选人）', '/owner/task/t1'],
        ['订单详情（已预约）', '/owner/order/t2'],
        ['我的订单', '/owner/orders'],
        ['我的', '/owner/me'],
    ],
    student: [
        ['任务大厅', '/student'],
        ['报价面板', '/student/task/h1'],
        ['我的任务', '/student/my'],
        ['上门履约', '/student/fulfill/t2'],
        ['收益中心', '/student/earnings'],
        ['学生认证 / 我的', '/student/verify'],
    ],
}

export default function SidePanel({role, pathname, onGo}) {
    return (
        <aside className="hidden w-[300px] shrink-0 lg:block">
            <div className="flex items-center gap-3">
                <span className="flex h-12 w-12 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-400 to-primary-600 text-white shadow-lg shadow-primary-500/30">
                    <PawPrint size={26} />
                </span>
                <div>
                    <h1 className="text-xl font-bold">上门喂宠 · 区域竞价</h1>
                    <p className="text-xs text-muted-foreground">小程序可点击原型 · V1.0 试点</p>
                </div>
            </div>

            <div className="mt-8 flex rounded-full bg-white/70 p-1 shadow-sm">
                {[
                    ['owner', '宠物主人端', UserRound, '/owner'],
                    ['student', '学生接单端', GraduationCap, '/student'],
                ].map(([key, label, Icon, path]) => (
                    <button
                        key={key}
                        type="button"
                        onClick={() => onGo(path)}
                        className={cn(
                            'flex flex-1 items-center justify-center gap-1.5 rounded-full py-2 text-sm font-semibold transition',
                            role === key ? 'bg-primary-500 text-white shadow' : 'text-muted-foreground',
                        )}
                    >
                        <Icon size={15} />
                        {label}
                    </button>
                ))}
            </div>

            <p className="mb-2 mt-6 text-xs font-semibold text-muted-foreground">页面快速跳转</p>
            <div className="grid grid-cols-2 gap-2">
                {PAGES[role].map(([label, path]) => (
                    <button
                        key={path}
                        type="button"
                        onClick={() => onGo(path)}
                        className={cn(
                            'rounded-xl border px-3 py-2.5 text-left text-xs font-medium transition hover:-translate-y-0.5 hover:shadow',
                            pathname === path
                                ? 'border-primary-400 bg-white text-primary-700'
                                : 'border-white/60 bg-white/60 text-foreground',
                        )}
                    >
                        {label}
                    </button>
                ))}
            </div>

            <div className="mt-6 rounded-2xl bg-white/70 p-4 text-xs leading-relaxed text-muted-foreground">
                <b className="text-foreground">完整闭环体验：</b>
                主人发布任务 → 切到学生端报价 → 回主人端选人并支付 → 学生签到、上传凭证 → 主人验收评价。
            </div>
        </aside>
    )
}
