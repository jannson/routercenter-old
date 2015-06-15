package rcenter

const (
	SEQ_START = 0
)

type SeqData interface {
	GetRequestId() int
	SetRequestId(seq int)
}

type SeqMap struct {
	index2obj []SeqData
	seq2index []int
	maxSize   int
	pos       int
	curr_seq  int
}

func NewSeqMap(maxSize int) *SeqMap {
	seqMap := &SeqMap{index2obj: make([]SeqData, maxSize), seq2index: make([]int, maxSize), maxSize: maxSize, pos: 0, curr_seq: 1}
	for i := 0; i < maxSize; i++ {
		seqMap.seq2index[i] = -1
		seqMap.index2obj[i] = nil
	}

	return seqMap
}

func (seqMap *SeqMap) NewSeq(data SeqData) SeqData {
	seq := seqMap.curr_seq
	if seq >= seqMap.maxSize {
		seq = SEQ_START
	}

	if seqMap.pos >= seqMap.maxSize {
		panic("exceed pos size")
	}

	index := seqMap.seq2index[seq]
	var oldData SeqData
	if index >= 0 {
		//Remove old seq
		oldData = seqMap.index2obj[index]
		seqMap.seq2index[oldData.GetRequestId()] = -1
	}

	data.SetRequestId(seq)
	seqMap.seq2index[seq] = seqMap.pos
	seqMap.index2obj[seqMap.pos] = data
	seqMap.curr_seq = seq + 1
	seqMap.pos += 1

	return oldData
}

func (seqMap *SeqMap) DelSeq(seq int) SeqData {
	if seq < SEQ_START || seq >= seqMap.maxSize {
		return nil
	}

	index := seqMap.seq2index[seq]
	if index < 0 {
		return nil
	}
	data := seqMap.index2obj[index]
	if data == nil || data.GetRequestId() != seq {
		return nil
	}

	seqMap.pos -= 1
	if seqMap.pos < 0 {
		panic("too small for pos size")
	}

	seqMap.seq2index[seq] = -1
	if index == seqMap.pos {
		seqMap.index2obj[index] = nil
	} else {
		//Set last pos to current deleted pos
		seqMap.index2obj[index] = seqMap.index2obj[seqMap.pos]
		seqMap.seq2index[seqMap.index2obj[seqMap.pos].GetRequestId()] = index
		seqMap.index2obj[seqMap.pos] = nil
	}

	return data
}

func (seqMap *SeqMap) GetData(seq int) SeqData {
	if seq < SEQ_START || seq >= seqMap.maxSize {
		return nil
	}

	index := seqMap.seq2index[seq]
	if index < 0 {
		return nil
	}
	data := seqMap.index2obj[index]
	if data == nil || data.GetRequestId() != seq {
		return nil
	}

	return data
}
