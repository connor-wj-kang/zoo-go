import {useEffect, useRef} from 'react'
import {Outlet, useLocation, useNavigate} from 'react-router-dom'
import {motion} from 'framer-motion'
import {PawPrint, Signal, Wifi, BatteryFull} from 'lucide-react'
import TabBar, {TABS} from '@/components/TabBar'
import SidePanel from '@/components/SidePanel'
import {useApp} from '@/store/AppContext'

function StatusBar() {
    return (
        <div className="pointer-events-none absolute inset-x-0 top-0 z-40 flex h-11 items-center justify-between px-7 pt-1 text-[13px] font-semibold">
            <span>9:41</span>
            <div className="flex items-center gap-1">
                <Signal size={14} />
                <Wifi size={14} />
                <BatteryFull size={18} />
            </div>
        </div>
    )
}

function Capsule() {
    return (
        <div className="pointer-events-none absolute right-3 top-[50px] z-40 flex h-[30px] w-[84px] items-center justify-around rounded-full border border-border bg-white/80 backdrop-blur">
            <span className="flex gap-[3px]">
                <i className="h-1 w-1 rounded-full bg-foreground" />
                <i className="h-1 w-1 rounded-full bg-foreground" />
                <i className="h-1 w-1 rounded-full bg-foreground" />
            </span>
            <span className="h-4 w-px bg-border" />
            <span className="flex h-4 w-4 items-center justify-center rounded-full border-2 border-foreground">
                <i className="h-1.5 w-1.5 rounded-full bg-foreground" />
            </span>
        </div>
    )
}

export default function PhoneLayout() {
    const {pathname} = useLocation()
    const navigate = useNavigate()
    const {toastMsg} = useApp()
    const scrollRef = useRef(null)
    const role = pathname.startsWith('/student') ? 'student' : 'owner'
    const showTab = TABS[role].some((t) => t.to === pathname)

    useEffect(() => {
        scrollRef.current?.scrollTo({top: 0})
    }, [pathname])

    return (
        <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-primary-100 via-accent-50 to-secondary-100 lg:gap-16">
            <SidePanel role={role} pathname={pathname} onGo={navigate} />
            <div className="relative h-screen w-full overflow-hidden bg-muted [transform:translateZ(0)] lg:h-[844px] lg:w-[390px] lg:rounded-[48px] lg:border-[10px] lg:border-foreground lg:shadow-2xl">
                <StatusBar />
                <Capsule />
                <div className="flex h-full flex-col">
                    <div ref={scrollRef} className="no-scrollbar flex-1 overflow-y-auto">
                        <motion.div
                            key={pathname}
                            initial={{opacity: 0, x: 14}}
                            animate={{opacity: 1, x: 0}}
                            transition={{duration: 0.25}}
                            className="min-h-full pb-6"
                        >
                            <Outlet />
                        </motion.div>
                    </div>
                    {showTab && <TabBar role={role} />}
                </div>
                {toastMsg && (
                    <div className="pointer-events-none fixed inset-x-0 top-1/2 z-[60] flex justify-center">
                        <div className="flex animate-scale-in items-center gap-2 rounded-2xl bg-foreground/90 px-5 py-3 text-[13px] text-white shadow-xl">
                            <PawPrint size={15} className="text-primary-300" />
                            {toastMsg}
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}
