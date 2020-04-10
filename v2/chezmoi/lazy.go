package chezmoi

import "crypto/sha256"

// A lazyContents evaluates its contents lazily.
type lazyContents struct {
	contentsFunc   func() ([]byte, error)
	contents       []byte
	contentsErr    error
	contentsSHA256 []byte
}

// A lazyLinkname evaluates its linkname lazily.
type lazyLinkname struct {
	linknameFunc func() (string, error)
	linkname     string
	linknameErr  error
}

// Contents returns e's contents.
func (lc *lazyContents) Contents() ([]byte, error) {
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
func (lc *lazyContents) ContentsSHA256() ([]byte, error) {
	if lc == nil {
		return sha256Sum(nil), nil
	}
	if lc.contentsSHA256 == nil {
		contents, err := lc.Contents()
		if err != nil {
			return nil, err
		}
		lc.contentsSHA256 = sha256Sum(contents)
	}
	return lc.contentsSHA256, nil
}

// Linkname returns s's linkname.
func (ll *lazyLinkname) Linkname() (string, error) {
	if ll == nil {
		return "", nil
	}
	if ll.linknameFunc != nil {
		ll.linkname, ll.linknameErr = ll.linknameFunc()
		ll.linknameFunc = nil
	}
	return ll.linkname, ll.linknameErr
}

func sha256Sum(data []byte) []byte {
	sha256SumArr := sha256.Sum256(data)
	return sha256SumArr[:]
}
