# Sourced by demo.tape. Provides a mock `clier` that emits canned output
# so the GIF can illustrate the hello-claude flow without real side effects
# (browser auth, tmux, vendor CLIs).

ATTACH_COUNT=0

clier() {
  local cmd="$*"
  case "$cmd" in
    "auth status")
      echo "Not logged in."
      ;;
    "auth login")
      echo "→ Open https://github.com/login/device and enter CODE-1234"
      sleep 0.7
      echo "✓ Logged in as alice"
      ;;
    "clone @clier/hello-claude")
      echo "✓ Cloned to ~/.clier/workspace/@clier/hello-claude"
      ;;
    "run start @clier/hello-claude")
      echo "run_id:  r-abc"
      echo "status:  launched"
      echo "hint:    first run — attach once to approve vendor prompts, then Ctrl-b d"
      ;;
    *)
      if [[ "$cmd" == run\ attach* ]]; then
        ATTACH_COUNT=$((ATTACH_COUNT + 1))
        if [ $ATTACH_COUNT -eq 1 ]; then
          echo "[tmux] hello-claude ● hello-codex ○"
          sleep 0.5
          echo "Codex: trust this directory? [y/N] y"
          sleep 0.4
          echo "(detached: Ctrl-b d)"
        else
          echo "[hello-claude] hi codex, ready to greet?"
          sleep 0.5
          echo "[hello-codex]  hi claude! ✓"
          sleep 0.5
          echo "[hello-claude] ✓ greeting exchange complete"
        fi
      elif [[ "$cmd" == run\ tell* ]]; then
        echo "✓ sent"
      else
        echo "clier: unknown demo command: $cmd" >&2
      fi
      ;;
  esac
}
