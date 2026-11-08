import {ChevronLeft} from 'lucide-react'
import {useNavigate} from 'react-router-dom'

export default function NavBar({title, back = true, right = null, backTo}) {
    const navigate = useNavigate()
    const goBack = () => (backTo ? navigate(backTo) : navigate(-1))
    return (
        <header className="sticky top-0 z-30 bg-white/90 pt-11 backdrop-blur">
            <div className="relative flex h-11 items-center px-3">
                {back ? (
                    <button
                        type="button"
                        onClick={goBack}
                        className="-ml-1 flex h-8 w-8 items-center justify-center rounded-full active:bg-muted"
                        aria-label="返回"
                    >
                        <ChevronLeft size={22} />
                    </button>
                ) : (
                    <span className="text-base font-bold">{title}</span>
                )}
                {back && (
                    <h1 className="pointer-events-none absolute inset-x-0 text-center text-[15px] font-semibold">
                        {title}
                    </h1>
                )}
                <div className="ml-auto mr-[92px]">{right}</div>
            </div>
        </header>
    )
}
