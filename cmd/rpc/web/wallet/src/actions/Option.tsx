import {cx} from "@/ui/cx";

export type OptionItem = { label: string; value: string; help?: string; icon?: string; toolTip?: string }

export const Option: React.FC<{
    selected: boolean
    disabled?: boolean
    onSelect: () => void
    label: React.ReactNode
    help?: React.ReactNode,
}> = ({ selected, disabled, onSelect, label, help }) => (
    <button
        type="button"
        onClick={onSelect}
        disabled={disabled}
        className={cx(
            "w-full rounded-md border p-3 text-left transition-colors",
            "focus:outline-none focus:ring-2 focus:ring-[#35cd48]/20",
            selected
                ? "border-[#35cd48]/35 bg-[#35cd48]/10"
                : "border-[#272729] bg-[#171717] hover:border-white/10 hover:bg-[#272729]/60",
            disabled && "opacity-60 cursor-not-allowed"
        )}
        aria-pressed={selected}
    >
        <div className="flex items-start gap-3">
      <span
          className={cx(
              "relative mt-1 h-4 w-4 rounded-full border",
              selected ? "border-[#35cd48]" : "border-[#272729]"
          )}
          aria-hidden
      >
          <span className={cx(
              "absolute left-1/2 top-1/2 h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full transition-colors",
              selected ? "bg-[#35cd48]" : "bg-transparent"
          )}/>
      </span>
            <div className="flex-1">
                <div className={cx("font-medium", selected ? "text-[#35cd48]" : "text-foreground")}>{label}</div>
                {help ? <div className={cx("mt-0.5 text-xs", selected ? "text-[#35cd48]/80" : "text-muted-foreground")}>{help}</div> : null}
            </div>
        </div>
    </button>
);
