# jirascrap e2e demo
# Exercises: list → select → detail → tag → todo → back → refresh → debug

Output e2e/demo.gif
Set Shell bash
Set FontSize 14
Set Width 1200
Set Height 700
Set Padding 10
Set Theme "Dracula"

# Start the app (env vars set by run.sh)
Type "jirascrap"
Enter
Sleep 1s

# --- List view ---
# Scroll down the ticket list
Sleep 1s
Down
Sleep 100ms
Down
Sleep 100ms
Down
Sleep 100ms

# --- Select ticket → Detail view ---
Up
Up
Sleep 200ms
Enter
Sleep 2s

# --- Tag the ticket ---
Type "t"
Sleep 1s
Type "backend, auth"
Sleep 500ms
Enter
Sleep 1s

# --- Open todo popup ---
Type "n"
Sleep 1s

# Add a todo item
Type "a"
Sleep 500ms
Type "Extend session TTL"
Enter
Sleep 500ms

# Add another
Type "a"
Sleep 500ms
Type "Add refresh token"
Enter
Sleep 500ms

# Toggle first todo done
Up
Sleep 300ms
Space
Sleep 500ms

# Close todo popup
Escape
Sleep 1s

# --- Go back to list ---
Escape
Sleep 1s

# --- Refresh from Jira ---
Type "r"
Sleep 3s

# --- Open debug overlay ---
Type "d"
Sleep 2s

# Close debug
Escape
Sleep 1s

# --- Quit ---
Ctrl+c
Sleep 1s
