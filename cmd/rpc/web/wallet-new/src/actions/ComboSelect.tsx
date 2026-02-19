// ComboSelect.tsx — asigna un valor libre y lo muestra como “opción extra” seleccionada
// (MISMO DISEÑO: mismas clases y tokens que tu versión)
"use client";

import * as React from "react";
import * as Popover from "@radix-ui/react-popover";
import * as ScrollArea from "@radix-ui/react-scroll-area";
import {ArrowRight, Check, ChevronsUpDown} from "lucide-react";
import {cx} from "@/ui/cx";

export type ComboOption = { label: string; value: string; disabled?: boolean };

export type ComboSelectProps = {
    id?: string;
    value?: string | null;
    options: ComboOption[];
    onChange: (val: string | null, meta?: { assigned?: boolean }) => void;

    placeholder?: string;
    emptyText?: string;
    disabled?: boolean;

    /** Permite asignar el texto escrito como valor del select (sin crearlo en la lista). */
    allowAssign?: boolean;
    /** Enter confirma el texto aunque no esté en options (atajo de teclado). */
    allowFreeInput?: boolean;

    // Estilo
    className?: string;        // Popover.Content
    buttonClassName?: string;  // Trigger
    listHeight?: number;       // px
};

export default function ComboSelect({
                                        id,
                                        value,
                                        options,
                                        onChange,
                                        placeholder = "Select",
                                        emptyText = "No results",
                                        disabled,
                                        allowAssign = true,
                                        allowFreeInput = true,
                                        className,
                                        buttonClassName,
                                        listHeight = 240,
                                    }: ComboSelectProps) {
    const [open, setOpen] = React.useState(false);
    const [query, setQuery] = React.useState("");
    const inputRef = React.useRef<HTMLInputElement>(null);
    const isClosingRef = React.useRef(false);

    // 🔹 Opción temporal “extra” cuando se asigna un valor libre
    const [tempOption, setTempOption] = React.useState<ComboOption | null>(null);

    // Si `value` viene de fuera y no existe en options, crea/actualiza tempOption para que se vea seleccionada
    React.useEffect(() => {
        if (!value) {
            if (tempOption) setTempOption(null);
            return;
        }
        const exists = options.some((o) => o.value === value);
        if (!exists) {
            setTempOption({value, label: value});
        } else if (tempOption && tempOption.value !== value) {
            setTempOption(null);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [value, options]);

    // Lista a renderizar = options + tempOption (si aplica). No mutamos la original.
    const mergedOptions = React.useMemo(() => {
        if (tempOption && !options.some((o) => o.value === tempOption.value)) {
            return [...options, tempOption];
        }
        return options;
    }, [options, tempOption]);

    const selected = mergedOptions.find((o) => o.value === value) || null;

    const filtered = React.useMemo(() => {
        const q = query.trim().toLowerCase();
        if (!q) return mergedOptions;
        return mergedOptions.filter((o) => (o.label + " " + o.value).toLowerCase().includes(q));
    }, [mergedOptions, query]);

    const closePopover = React.useCallback(() => {
        if (isClosingRef.current) return;
        isClosingRef.current = true;
        setOpen(false);
        setQuery("");
        setTimeout(() => {
            isClosingRef.current = false;
        }, 100);
    }, []);

    const assignValue = (text: string) => {
        const v = text.trim();
        if (!v) return;
        // Creamos/actualizamos la opción temporal y la seleccionamos
        const opt = {value: v, label: v};
        setTempOption(opt);
        onChange(v, {assigned: true}); // <- solo asigna; no persiste en options global
        closePopover();
    };

    const handlePick = (val: string) => {
        onChange(val, {assigned: false});
        closePopover();
    };

    const onKeyDown: React.KeyboardEventHandler<HTMLInputElement> = (e) => {
        if (e.key === "Enter" && query.trim() && allowFreeInput && allowAssign) {
            e.preventDefault();
            assignValue(query);
        }
        if (e.key === "Escape") {
            e.preventDefault();
            e.stopPropagation();
            closePopover();
        }
    };

    return (
        <Popover.Root
            open={open}
            modal={true}
            onOpenChange={(o) => {
                if (!o) {
                    closePopover();
                } else {
                    if (!isClosingRef.current) {
                        setOpen(true);
                        setTimeout(() => inputRef.current?.focus(), 50);
                    }
                }
            }}
        >
            <Popover.Trigger asChild>
                <button
                    id={id}
                    type="button"
                    disabled={disabled}
                    className={
                        buttonClassName ??
                        "flex items-center justify-between whitespace-nowrap border px-3 py-2 text-sm shadow-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1 w-full bg-muted border-border text-foreground h-11 rounded-lg"
                    }
                    aria-haspopup="listbox"
                    aria-expanded={open}
                >
          <span className={!selected ? "text-muted-foreground truncate" : "truncate"}>
            {selected ? selected.label : placeholder}
          </span>
                    <ChevronsUpDown className="h-4 w-4 opacity-60"/>
                </button>
            </Popover.Trigger>

            <Popover.Content
                sideOffset={6}
                align="start"
                onInteractOutside={(e) => {
                    // Prevent closing when clicking on the trigger
                    const target = e.target as HTMLElement;
                    if (target.closest('[role="combobox"]')) {
                        e.preventDefault();
                        return;
                    }
                    closePopover();
                }}
                onEscapeKeyDown={(e) => {
                    e.preventDefault();
                    closePopover();
                }}
                className={
                    className ??
                    "z-50 w-[--radix-popover-trigger-width] min-w-56 rounded-xl p-2 shadow-xl bg-muted border border-border"
                }
            >
                {/* Input */}
                <div className="flex items-center gap-2 border-b px-2 py-1.5 text-canopy-50">
                    <input
                        ref={inputRef}
                        value={query}
                        onChange={(e) => setQuery(e.target.value)}
                        onKeyDown={onKeyDown}
                        placeholder={placeholder}
                        className="w-full bg-transparent outline-none placeholder:text-muted-foreground"
                    />
                </div>

                <div className="mt-2 text-sm">
                    {filtered.length === 0 && (
                        <div className="px-2 py-2 text-muted-foreground">{emptyText}</div>
                    )}

                    {filtered.length > 0 && (
                        <ScrollArea.Root className="overflow-hidden rounded-lg" style={{maxHeight: listHeight}}>
                            <ScrollArea.Viewport className="p-1">
                                <ul role="listbox">
                                    {filtered.map((opt) => {
                                        const isSel = value === opt.value;
                                        return (
                                            <li key={opt.value} role="option" aria-selected={isSel}>
                                                <button
                                                    type="button"
                                                    disabled={opt.disabled}
                                                    onClick={() => handlePick(opt.value)}
                                                    className={cx(
                                                        "flex w-full items-center gap-2 rounded-md px-2 py-2 text-left hover:bg-primary hover:text-primary-foreground disabled:opacity-50 text-canopy-50 mt-1",
                                                        isSel && "bg-primary text-primary-foreground"
                                                    )}
                                                >
                                                    <Check
                                                        className={cx("h-4 w-4", isSel ? "opacity-100" : "opacity-0")}/>
                                                    <span className={cx("truncate")}>
                                                        {opt.label}
                                                    </span>
                                                </button>
                                            </li>
                                        );
                                    })}
                                </ul>
                            </ScrollArea.Viewport>
                            <ScrollArea.Scrollbar orientation="vertical" className="flex select-none touch-none p-0.5">
                                <ScrollArea.Thumb className="flex-1 rounded bg-muted-foreground/40"/>
                            </ScrollArea.Scrollbar>
                        </ScrollArea.Root>
                    )}

                    {allowAssign && query.trim() && (
                        <div className="pt-2">
                            <button
                                type="button"
                                onClick={() => assignValue(query)}
                                className="flex w-full items-center gap-2 rounded-md px-2 py-2 hover:bg-primary text-canopy-50 hover:text-primary-foreground"
                            >
                                <ArrowRight className="h-4 w-4"/>
                                Assign “{query.trim()}”
                            </button>
                        </div>
                    )}
                </div>
            </Popover.Content>
        </Popover.Root>
    );
}

