package chezmoi

func newLazyContents(contents []byte) *lazyContents {
	return &lazyContents{
		contentsFunc: func() ([]byte, error) {
			return contents, nil
		},
	}
}

func newLazyLinkname(linkname string) *lazyLinkname {
	return &lazyLinkname{
		linknameFunc: func() (string, error) {
			return linkname, nil
		},
	}
}
