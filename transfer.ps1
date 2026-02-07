Import-Module Posh-SSH
$secpasswd = ConvertTo-SecureString "Pi16870403" -AsPlainText -Force
$cred = New-Object System.Management.Automation.PSCredential ("pi", $secpasswd)

$loc = "c:\Users\victor.s.heredia\OneDrive - Avanade\Documents\Proyectos\CoE\Claude-Code\dockerverse\frontend"
$dest = "/home/pi/dockerverse/frontend"
$ip = "192.168.1.145"

$files = @(
    @{ src = "src\lib\stores\auth.ts"; dst = "src/lib/stores/" },
    @{ src = "src\lib\stores\docker.ts"; dst = "src/lib/stores/" },
    @{ src = "src\lib\components\Login.svelte"; dst = "src/lib/components/" },
    @{ src = "src\lib\components\Settings.svelte"; dst = "src/lib/components/" },
    @{ src = "src\lib\components\ContainerCard.svelte"; dst = "src/lib/components/" },
    @{ src = "src\lib\components\LogViewer.svelte"; dst = "src/lib/components/" },
    @{ src = "src\lib\components\Terminal.svelte"; dst = "src/lib/components/" },
    @{ src = "src\lib\components\index.ts"; dst = "src/lib/components/" },
    @{ src = "src\lib\api\docker.ts"; dst = "src/lib/api/" },
    @{ src = "src\routes\+layout.svelte"; dst = "src/routes/" },
    @{ src = "src\routes\+page.svelte"; dst = "src/routes/" }
)

foreach ($f in $files) {
    $srcPath = Join-Path $loc $f.src
    $dstPath = "$dest/$($f.dst)"
    Write-Host "Transferring: $($f.src)..." -NoNewline
    Set-SCPItem -ComputerName $ip -Credential $cred -Path $srcPath -Destination $dstPath -Force -ErrorAction Stop
    Write-Host " OK"
}

Write-Host "`nAll files transferred!"

# Now rebuild
Write-Host "`nRebuilding frontend..."
$session = New-SSHSession -ComputerName $ip -Credential $cred -Force
$build = Invoke-SSHCommand -SessionId $session.SessionId -Command "cd ~/dockerverse && docker compose build frontend --no-cache 2>&1" -Timeout 300
Write-Host "Build exit code: $($build.ExitStatus)"
$build.Output | Select-Object -Last 10

# Restart
Write-Host "`nRestarting frontend..."
$restart = Invoke-SSHCommand -SessionId $session.SessionId -Command "cd ~/dockerverse && docker compose up -d frontend 2>&1 && sleep 3 && docker compose ps" -Timeout 60
$restart.Output

Remove-SSHSession -SessionId $session.SessionId

Write-Host "`nDone! Access at http://192.168.1.145:3002"
