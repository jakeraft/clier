# Clier hello-claude demo
# Regenerate: vhs demo/demo.tape  (from repo root)

Output demo/demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1100
Set Height 640
Set TypingSpeed 35ms
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set Padding 20

Hide
Type "source demo/demo.sh" Enter
Type `export PS1='$ '` Enter
Type "clear" Enter
Show

Sleep 600ms

Type "# your CLI agent begins..." Enter
Sleep 500ms

Type "clier auth status" Enter
Sleep 1200ms

Type `# agent: "Please run 'clier auth login'"` Enter
Sleep 1500ms

Type "clier auth login" Enter
Sleep 2200ms

Type "clier clone @clier/hello-claude" Enter
Sleep 1000ms

Type "clier run start @clier/hello-claude" Enter
Sleep 1500ms

Type `# agent: "Attach, approve vendor prompts, then Ctrl-b d"` Enter
Sleep 1800ms

Type "clier run attach r-abc" Enter
Sleep 2000ms

Type `clier run tell --run r-abc --to @clier/hello-claude "greet each other"` Enter
Sleep 1200ms

Type "clier run attach r-abc  # watch live" Enter
Sleep 2500ms
