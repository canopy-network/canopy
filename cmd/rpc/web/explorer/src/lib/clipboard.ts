export async function copyText(text: string): Promise<void> {
    if (window.isSecureContext && navigator.clipboard) {
        try {
            await navigator.clipboard.writeText(text)
            return
        } catch {
            // Try the fallback when clipboard permissions are denied.
        }
    }

    // The Clipboard API is unavailable on HTTP LAN addresses.
    const textarea = document.createElement('textarea')
    const activeElement = document.activeElement
    textarea.value = text
    textarea.contentEditable = 'true'
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    try {
        textarea.select()
        // iOS needs an editable element and an explicit selection range.
        const range = document.createRange()
        range.selectNodeContents(textarea)
        const selection = window.getSelection()
        selection?.removeAllRanges()
        selection?.addRange(range)
        textarea.setSelectionRange(0, text.length)
        if (!document.execCommand('copy')) {
            throw new Error('Unable to copy text')
        }
    } finally {
        textarea.remove()
        if (activeElement instanceof HTMLElement) {
            activeElement.focus({ preventScroll: true })
        }
    }
}
