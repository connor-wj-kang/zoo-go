import {cn} from '@/utils/cn'

export default function ChipGroup({options, value, onChange, multiple = false, getLabel = (o) => o}) {
    const isOn = (o) => (multiple ? value.includes(o) : value === o)
    const toggle = (o) => {
        if (!multiple) {
            onChange(o)
            return
        }
        onChange(value.includes(o) ? value.filter((v) => v !== o) : [...value, o])
    }
    return (
        <div className="flex flex-wrap gap-2">
            {options.map((o) => (
                <button key={getLabel(o)} type="button" onClick={() => toggle(o)} className={cn('chip', isOn(o) && 'chip-active')}>
                    {getLabel(o)}
                </button>
            ))}
        </div>
    )
}
