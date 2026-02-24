# shelfctl TUI demo

Output tui_demo.gif

Set Shell "zsh"
Set TypingSpeed 10ms
Set FontSize 14
Set FontFamily "MesloLGS NF"
Set Width 1200
Set Height 800
Set Padding 20
Set BorderRadius 8
Set WindowBar Colorful

Hide
Type "source ~/.zshrc && export PATH=/Users/dayna.blackwell/code/shelfctl/bin:$PATH"
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 300ms
Show

# ── 0. Cache clear to show lets-go is uncached ────────────────────────────────
Type "shelfctl cache clear lets-go --force"
Enter
Sleep 2000ms

# ── 1. Hub ────────────────────────────────────────────────────────────────────
Type "shelfctl"
Enter
Sleep 4000ms

# ── 2. Browse ─────────────────────────────────────────────────────────────────
Enter
Sleep 1500ms

Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms

# Download uncached book
Type "g"
Sleep 2500ms

# Show details tab
Tab
Sleep 1500ms

# Search
Type "/"
Sleep 600ms
Type "go"
Sleep 2000ms
Escape
Sleep 800ms

# Back to hub
Escape
Sleep 1500ms

# ── 3. Navigate to Edit Book (Add Book > Add from URL > Edit Book = 3 downs) ──
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Enter
Sleep 1500ms

# ── 4. Multi-select picker — select 5 books ───────────────────────────────────
Space
Sleep 400ms
Down
Sleep 300ms
Space
Sleep 400ms
Down
Sleep 300ms
Space
Sleep 400ms
Down
Sleep 300ms
Space
Sleep 400ms
Down
Sleep 300ms
Space
Sleep 400ms
Enter
Sleep 1500ms

# ── 5. Carousel — leftmost card (left ghost visible) ─────────────────────────
Sleep 2000ms

Right
Sleep 400ms
Right
Sleep 400ms
Right
Sleep 400ms
Right
Sleep 750ms

# Rightmost card — right ghost visible, pause longer
Sleep 500ms

# Pan back left
Left
Sleep 400ms
Left
Sleep 400ms

# ── 6. Drop into edit form ────────────────────────────────────────────────────
Enter
Sleep 2000ms

# Navigate down through fields
Down
Sleep 700ms
Down
Sleep 700ms
Down
Sleep 700ms
Down
Sleep 1500ms

# ── 7. Esc back to hub as final frame ─────────────────────────────────────────
Escape
Sleep 2000ms
