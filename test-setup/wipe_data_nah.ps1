# Specify the directories where you want to remove files
$directoriesToRemoveFilesFrom = @(
    "C:\Users\edmun\Documents\blockchain\blockchain\test-setup\node1_data",
    "C:\Users\edmun\Documents\blockchain\blockchain\test-setup\node2_data",
    "C:\Users\edmun\Documents\blockchain\blockchain\test-setup\node3_data",
    "C:\Users\edmun\Documents\blockchain\blockchain\test-setup\node4_data",
    "C:\Users\edmun\Documents\blockchain\blockchain\test-setup\node5_data"
    # Add more directories as needed
)

# Specify the names of the files you want to remove
$filesToRemove = @("blockchain.txt", "sorted_messages.txt")

# Loop through each directory
foreach ($directory in $directoriesToRemoveFilesFrom) {
    Write-Host "Processing directory: $directory"
    
    # Loop through each file to remove
    foreach ($file in $filesToRemove) {
        $filePath = Join-Path -Path $directory -ChildPath $file
        if (Test-Path $filePath) {
            Remove-Item -Path $filePath -Force
            Write-Host "Removed file: $file"
        } else {
            Write-Host "File not found: $file"
        }
    }
}
