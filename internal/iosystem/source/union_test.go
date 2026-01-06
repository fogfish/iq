//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package source_test

/*

TODO: Move union to the level of Collector

func TestUnionSource(t *testing.T) {
	t.Run("Union/TwoFiles", func(t *testing.T) {
		// Create temp files
		tmpDir := t.TempDir()
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")
		it.Then(t).Should(
			it.Nil(os.WriteFile(file1, []byte("content1"), 0644)),
			it.Nil(os.WriteFile(file2, []byte("content2"), 0644)),
		)

		// Create filesystem
		fsys := os.DirFS(tmpDir)

		// Create sources
		src1, err := source.NewFile(fsys, "file1.txt")
		it.Then(t).Should(it.Nil(err))
		src2, err := source.NewFile(fsys, "file2.txt")
		it.Then(t).Should(it.Nil(err))

		// Create merged source
		merged, err := source.NewUnion(src1, src2)
		it.Then(t).Should(it.Nil(err))
		defer merged.Close()

		// Read merged document
		doc, err := merged.Next(context.Background())
		it.Then(t).Should(it.Nil(err))

		content, err := io.ReadAll(doc.Reader)
		it.Then(t).Should(it.Nil(err))

		// Should contain both contents with newline separator
		result := string(content)
		it.Then(t).Should(
			it.String(result).Contain("content1"),
			it.String(result).Contain("content2"),
		)
	})

	t.Run("Union/SecondCallReturnsEOF", func(t *testing.T) {
		// Create temp file
		tmpDir := t.TempDir()
		file1 := filepath.Join(tmpDir, "file1.txt")
		it.Then(t).Should(
			it.Nil(os.WriteFile(file1, []byte("content"), 0644)),
		)

		// Create filesystem
		fsys := os.DirFS(tmpDir)

		// Create source
		src1, _ := source.NewFile(fsys, "file1.txt")

		// Create merged source
		merged, err := source.NewUnion(src1)
		it.Then(t).Should(it.Nil(err))
		defer merged.Close()

		ctx := context.Background()

		// First call should succeed
		doc, err := merged.Next(ctx)
		it.Then(t).Should(
			it.Nil(err),
			it.True(doc != nil),
		)

		// Second call should return EOF
		_, err = merged.Next(ctx)
		it.Then(t).Should(
			it.Equal(err, io.EOF),
		)
	})

	t.Run("Union/EmptySource", func(t *testing.T) {
		// Create temp directory with no files
		tmpDir := t.TempDir()
		fsys := os.DirFS(tmpDir)

		// Create file source with empty directory (no files)
		// This will return EOF immediately
		src, err := source.NewFile(fsys, "nonexistent.txt")
		it.Then(t).Should(it.Nil(err))

		// Create merged source
		merged, err := source.NewUnion(src)
		it.Then(t).Should(it.Nil(err))
		defer merged.Close()

		// Read merged document - should get error since file doesn't exist
		ctx := context.Background()
		_, err = merged.Next(ctx)
		it.Then(t).ShouldNot(it.Nil(err))
	})

	t.Run("ErrorNoSources", func(t *testing.T) {
		// Try to create merged source with no sources
		merged, err := source.NewUnion()
		it.Then(t).ShouldNot(
			it.Nil(err),
		)
		it.Then(t).Should(
			it.Nil(merged),
		)
	})
}

*/
