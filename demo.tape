Output demo.gif

Set Shell "zsh"
Set FontSize 15
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 40ms
Set WindowBar Colorful
Set WindowBarSize 40

# Step 1: Ping
Type "mysh ping production"
Enter
Sleep 2s

# Step 2: Query (masked – production env)
Type "mysh run production -e 'SELECT * FROM users'"
Enter
Sleep 4s

# Step 3: List tables
Type "mysh tables production"
Enter
Sleep 3s

# Step 4: JSON output
Type "mysh run production -e 'SELECT * FROM users LIMIT 2' --format json"
Enter
Sleep 4s

Sleep 1s
