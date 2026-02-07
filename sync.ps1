$cred = New-Object System.Management.Automation.PSCredential("pi", (ConvertTo-SecureString "Pi16870403" -AsPlainText -Force))
$base = "c:\Users\victor.s.heredia\OneDrive - Avanade\Documents\Proyectos\CoE\Claude-Code\dockerverse\frontend"
$files = @(
    "src\lib\api\docker.ts",
    "src\routes\+page.svelte",
    "src\routes\+layout.svelte",
    "src\lib\stores\docker.ts",
    "src\lib\components\ContainerCard.svelte",
    "src\lib\components\HostCard.svelte",
    "src\lib\components\Terminal.svelte",
    "src\lib\components\LogViewer.svelte"
)
foreach ($f in $files) {
    $local = "$base\$f"
    $dest = "/home/pi/dockerverse/frontend/" + ($f -replace '\\\\', '/')
    $destDir = $dest -replace '/[^/]+$', ''
    Set-SCPItem -ComputerName "192.168.1.145" -Credential $cred -Path $local -Destination "$destDir/" -Force
    Write-Host "$f OK"
}
