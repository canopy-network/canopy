import React from "react";

export default function useDebouncedValue<T>(value: T, delay = 250) {
    const [v, setV] = React.useState(value)
    React.useEffect(() => {
        const t = setTimeout(() => setV(value), delay)
        return () => clearTimeout(t)
    }, [value, delay])
    return v
}