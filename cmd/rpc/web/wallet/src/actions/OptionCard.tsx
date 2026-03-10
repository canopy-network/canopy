import {cx} from "@/ui/cx";

export type OptionCardOpt = { label: string; value: string; help?: string; icon?: string; toolTip?: string }

export const OptionCard: React.FC<{
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
            "w-full text-left rounded-md border-2 border-muted p-3 transition-colors",
            "focus:outline-none focus:ring-2 focus:ring-emerald-400",
            disabled && "opacity-60 cursor-not-allowed"
        )}
        aria-pressed={selected}
    >
        <div className="flex items-start gap-3">
      <span
          className={cx(
              "mt-1  h-4 w-4 rounded-full border relative",
              selected ? "border-emerald-400" : "border-border"
          )}
          aria-hidden
      >
          <span className={cx(selected && "bg-primary" , " absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 h-3 w-3 rounded-full")}/>
      </span>
            <div className="flex-1">
                <div className="font-medium text-canopy-50">{label}</div>
                {help ? <div className="text-xs text-muted-foreground mt-0.5">{help}</div> : null}
            </div>
        </div>
    </button>
);

