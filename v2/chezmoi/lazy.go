package chezmoi

import "crypto/sha256"

// A LazyContents evaluates its contents lazily.
type LazyContents struct {
	contentsFunc   func() ([]byte, error)
	contents       []byte
	contentsErr    error
	contentsSHA256 []byte
}

// A LazyLinkname evaluates its linkname lazily.
type LazyLinkname struct {
	linknameFunc func() (string, error)
	linkname     string
	linknameErr  error
}

// Contents returns e's contents.
func (lc *LazyContents) Contents() ([]byte, error) {
	if lc == nil {
		return nil, nil
	}
	if lc.contentsFunc != nil {
		lc.contents, lc.contentsErr = lc.contentsFunc()
		lc.contentsFunc = nil
		if lc.contentsErr == nil {
			lc.contentsSHA256 = sha256Sum(lc.contents)
		}
	}
	return lc.contents, lc.contentsErr
}

// ContentsSHA256 returns the SHA256 sum of f's contents.
func (lc *LazyContents) ContentsSHA256() ([]byte, error) {
	if lc == nil {
		return sha256Sum(nil), nil
	}
	if lc.contentsSHA256 == nil {
		if _, err := lc.Contents(); err != nil {
			return nil, err
		}
		lc.contentsSHA256 = sha256Sum(lc.contents)
	}
	return lc.contentsSHA256, nil
}

// Linkname returns s's linkname.
func (ll *LazyLinkname) Linkname() (string, error) {
	if ll == nil {
		return "", nil
	}
	if ll.linknameFunc != nil {
		ll.linkname, ll.linknameErr = ll.linknameFunc()
	}
	return ll.linkname, ll.linknameErr
}

func sha256Sum(data []byte) []byte {
	sha256SumArr := sha256.Sum256(data)
	return sha256SumArr[:]
}
