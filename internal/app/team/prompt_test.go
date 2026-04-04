package team

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestJoinPrompts(t *testing.T) {
	t.Run("SinglePrompt_ReturnsAsIs", func(t *testing.T) {
		prompts := []domain.PromptSnapshot{
			{Name: "style", Prompt: "Be concise."},
		}

		got := joinPrompts(prompts)
		if got != "Be concise." {
			t.Errorf("got %q, want %q", got, "Be concise.")
		}
	})

	t.Run("MultiplePrompts_JoinedWithSeparator", func(t *testing.T) {
		prompts := []domain.PromptSnapshot{
			{Name: "style", Prompt: "Be concise."},
			{Name: "role", Prompt: "You are a Go developer."},
			{Name: "rules", Prompt: "Follow best practices."},
		}

		got := joinPrompts(prompts)
		want := "Be concise.\n\n---\n\nYou are a Go developer.\n\n---\n\nFollow best practices."
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("NoPrompts_ReturnsEmpty", func(t *testing.T) {
		got := joinPrompts(nil)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("EmptySlice_ReturnsEmpty", func(t *testing.T) {
		got := joinPrompts([]domain.PromptSnapshot{})
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}
