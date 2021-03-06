package blob

import (
	"encoding/hex"
	"io"

	"github.com/anacrolix/libtorgo/metainfo"
)

type data struct {
	info  *metainfo.Info
	store *store
}

func (me *data) pieceHashHex(i int) string {
	return hex.EncodeToString(me.info.Pieces[i*20 : (i+1)*20])
}

func (me *data) Close() {}

func (me *data) ReadAt(b []byte, off int64) (n int, err error) {
	p := me.info.Piece(int(off / me.info.PieceLength))
	f := me.store.pieceRead(p)
	if f == nil {
		err = io.ErrUnexpectedEOF
		return
	}
	defer f.Close()
	n, err = f.ReadAt(b, off%me.info.PieceLength)
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return
}

func (me *data) WriteAt(p []byte, off int64) (n int, err error) {
	i := int(off / me.info.PieceLength)
	off %= me.info.PieceLength
	for len(p) != 0 {
		f := me.store.pieceWrite(me.info.Piece(i))
		p1 := p
		maxN := me.info.Piece(i).Length() - off
		if int64(len(p1)) > maxN {
			p1 = p1[:maxN]
		}
		var n1 int
		n1, err = f.WriteAt(p1, off)
		f.Close()
		n += n1
		if err != nil {
			return
		}
		p = p[n1:]
		off = 0
		i++
	}
	return
}

func (me *data) pieceReader(piece int, off int64) (ret io.ReadCloser, err error) {
	f := me.store.pieceRead(me.info.Piece(piece))
	if f == nil {
		err = io.ErrUnexpectedEOF
		return
	}
	return struct {
		io.Reader
		io.Closer
	}{
		Reader: io.NewSectionReader(f, off, me.info.Piece(piece).Length()-off),
		Closer: f,
	}, nil
}

func (me *data) WriteSectionTo(w io.Writer, off, n int64) (written int64, err error) {
	i := int(off / me.info.PieceLength)
	off %= me.info.PieceLength
	for n != 0 {
		var pr io.ReadCloser
		pr, err = me.pieceReader(i, off)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
			pr.Close()
		}
		var n1 int64
		n1, err = io.CopyN(w, pr, n)
		pr.Close()
		written += n1
		n -= n1
		if err != nil {
			return
		}
		off = 0
		i++
	}
	return
}

func (me *data) PieceCompleted(index int) (err error) {
	return me.store.PieceCompleted(me.info.Piece(index))
}

func (me *data) PieceComplete(piece int) bool {
	return me.store.pieceComplete(me.info.Piece(piece))
}
