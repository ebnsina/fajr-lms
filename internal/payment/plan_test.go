package payment

import "testing"

func TestPart(t *testing.T) {
	t.Run("the parts add up to the whole", func(t *testing.T) {
		for _, total := range []int64{100000, 99999, 3, 150000} {
			for _, parts := range []int{2, 3, 7} {
				var sum int64
				for part := 1; part <= parts; part++ {
					amount, err := Part(total, parts, part)
					if err != nil {
						if total == 3 && parts > 3 {
							continue
						}
						t.Fatalf("total %d in %d parts, part %d: %v", total, parts, part, err)
					}
					if amount < 1 {
						t.Fatalf("part %d of %d came to %d", part, parts, amount)
					}
					sum += amount
				}
				if sum != total && !(total == 3 && parts > 3) {
					t.Fatalf("%d in %d parts adds up to %d", total, parts, sum)
				}
			}
		}
	})

	t.Run("what cannot be split is refused", func(t *testing.T) {
		if _, err := Part(3, 7, 1); err == nil {
			t.Fatal("splitting 3 seven ways was allowed")
		}
		if _, err := Part(1000, 1, 1); err == nil {
			t.Fatal("a plan of one part was allowed")
		}
		if _, err := Part(1000, 4, 5); err == nil {
			t.Fatal("a part beyond the end of the plan was allowed")
		}
	})
}
