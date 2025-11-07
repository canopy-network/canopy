Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$img = [System.Windows.Forms.Clipboard]::GetImage()

if ($img -ne $null) {
    $img.Save("figma-design.png", [System.Drawing.Imaging.ImageFormat]::Png)
    Write-Host "Image saved to figma-design.png"
} else {
    Write-Host "No image found in clipboard"
}
