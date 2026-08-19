package pseudonym_test

import (
	"testing"

	pseudonympkg "github.com/iilei/nopii/internal/pseudonym"
)

func TestDeterministicAndScoped(t *testing.T) {
	g := pseudonympkg.New([]byte("secret"), "repo-a", 12)
	a := g.Replacement("PERSON", "Alice")
	b := g.Replacement("PERSON", "Alice")
	if a != b {
		t.Fatalf("same input changed: %q != %q", a, b)
	}
	if a == pseudonympkg.New([]byte("secret"), "repo-b", 12).Replacement("PERSON", "Alice") {
		t.Fatal("scope did not affect token")
	}
	if a == g.Replacement("PERSON", "Bob") {
		t.Fatal("different values collided")
	}
}

func TestEmailNormalization(t *testing.T) {
	g := pseudonympkg.New([]byte("secret"), "default", 12)
	if g.Replacement("EMAIL", "Alice@Example.COM") != g.Replacement("EMAIL", "alice@example.com") {
		t.Fatal("email normalization is not case insensitive")
	}
}

func TestDefaultAlgorithmMatchesExplicitV1(t *testing.T) {
	defaultGenerator := pseudonympkg.New([]byte("secret"), "default", 12)
	explicitGenerator := pseudonympkg.NewWithAlgorithm([]byte("secret"), "default", 12, "v1")
	got := defaultGenerator.Replacement("PERSON", "Alice")
	want := explicitGenerator.Replacement("PERSON", "Alice")
	if got != want {
		t.Fatalf("default algorithm changed: %q != %q", got, want)
	}
}
