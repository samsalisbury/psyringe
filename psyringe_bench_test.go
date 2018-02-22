package psyringe

import "testing"

var benchErr error

func BenchmarkPsyringe_Add(b *testing.B) {

	type (
		A *struct{}
		B *struct{}
		C *struct{}
		D *struct{}
		E *struct{}
		F *struct{}
	)

	exercise := func(p *Psyringe) {
		benchErr = p.AddErr(
			func(F) A { return nil },
			func(C) B { return nil },
			func(D) C { return nil },
			func(E) D { return nil },
			func(F) E { return nil },
			func(A) F { return nil },
		)
	}

	b.Run("with detectCycle", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p := New()
			p.allowAddCycle = false
			exercise(p)
		}
	})

	b.Run("without detectCycle", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			p := New()
			p.allowAddCycle = true
			exercise(p)
		}
	})
}
