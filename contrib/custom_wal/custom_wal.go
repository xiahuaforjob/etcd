package custom_wal

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"go.etcd.io/etcd/server/v3/storage/wal/walpb"
	"go.etcd.io/raft/v3"
	"go.etcd.io/raft/v3/raftpb"
	"go.uber.org/zap"
)

const (
	MetadataType int64 = iota + 1
	EntryType
	StateType
	CrcType
	SnapshotType
)

// mustUnmarshalEntry parses raftpb.Entry from byte data.
func mustUnmarshalEntry(data []byte) raftpb.Entry {
	var entry raftpb.Entry
	if err := entry.Unmarshal(data); err != nil {
		panic(fmt.Sprintf("failed to unmarshal entry: %v", err))
	}
	return entry
}

// mustUnmarshalState parses raftpb.HardState from byte data.
func mustUnmarshalState(data []byte) raftpb.HardState {
	var state raftpb.HardState
	if err := state.Unmarshal(data); err != nil {
		panic(fmt.Sprintf("failed to unmarshal state: %v", err))
	}
	return state
}

// encoder encodes records into a write-ahead log file.
type encoder struct {
	w   io.Writer
	mu  sync.Mutex
	buf []byte
}

// newEncoder creates a new encoder.
func newEncoder(w io.Writer) *encoder {
	return &encoder{w: w}
}

// encode writes a record to the encoder's buffer.
func (e *encoder) encode(rec *walpb.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, err := rec.Marshal()
	if err != nil {
		return err
	}
	e.buf = append(e.buf[:0], data...)
	if _, err := e.w.Write(e.buf); err != nil {
		return err
	}
	return nil
}

// flush flushes the encoder's buffer to the underlying writer.
func (e *encoder) flush() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, err := e.w.Write(e.buf); err != nil {
		return err
	}
	e.buf = e.buf[:0]
	return nil
}

// decoder decodes records from a write-ahead log file.
type decoder struct {
	r   io.Reader
	mu  sync.Mutex
	buf []byte
}

// newDecoder creates a new decoder.
func newDecoder(r io.Reader) *decoder {
	return &decoder{r: r}
}

// decode reads a record from the decoder.
func (d *decoder) decode() (*walpb.Record, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var rec walpb.Record
	if err := rec.Unmarshal(d.buf); err != nil {
		return nil, err
	}
	return &rec, nil
}

// WAL represents a custom write-ahead log implementation.
type WAL struct {
	logger  *zap.Logger
	dir     string
	file    *os.File
	mu      sync.Mutex
	encoder *encoder
}

// writeMetadata writes metadata to the WAL file.
func (w *WAL) writeMetadata(metadata []byte) error {
	if w.encoder == nil {
		w.encoder = newEncoder(w.file)
	}
	rec := &walpb.Record{
		Type: MetadataType,
		Data: metadata,
	}
	return w.encoder.encode(rec)
}

// Create initializes a new WAL file.
func Create(logger *zap.Logger, dir string, metadata []byte) (*WAL, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "wal.log")
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := &WAL{
		logger: logger,
		dir:    dir,
		file:   f,
	}
	if metadata != nil {
		if err := w.writeMetadata(metadata); err != nil {
			return nil, err
		}
	}
	return w, nil
}

// Open opens an existing WAL file.
func Open(logger *zap.Logger, dir string, snap walpb.Snapshot) (*WAL, error) {
	path := filepath.Join(dir, "wal.log")
	f, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	w := &WAL{
		logger: logger,
		dir:    dir,
		file:   f,
	}
	return w, nil
}

// ReadAll reads all entries from the WAL.
func (w *WAL) ReadAll() ([]byte, raftpb.HardState, []raftpb.Entry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, raftpb.HardState{}, nil, err
	}
	decoder := newDecoder(w.file)
	var metadata []byte
	var state raftpb.HardState
	var entries []raftpb.Entry
	for {
		rec, err := decoder.decode()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, raftpb.HardState{}, nil, err
		}
		switch rec.Type {
		case MetadataType:
			metadata = rec.Data
		case StateType:
			state = mustUnmarshalState(rec.Data)
		case EntryType:
			entries = append(entries, mustUnmarshalEntry(rec.Data))
		}
	}
	return metadata, state, entries, nil
}

// SaveSnapshot saves a snapshot to the WAL.
func (w *WAL) SaveSnapshot(snap walpb.Snapshot) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := snap.Marshal()
	if err != nil {
		return err
	}
	return w.encoder.encode(&walpb.Record{Type: SnapshotType, Data: data})
}

// ReleaseLockTo releases the WAL lock up to the specified index.
func (w *WAL) ReleaseLockTo(index uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return errors.New("WAL file is not open")
	}

	// For simplicity, we truncate the file to release space.
	// In a real implementation, you might need to handle metadata or snapshots.
	if err := w.file.Truncate(0); err != nil {
		return fmt.Errorf("failed to truncate WAL file: %v", err)
	}
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to reset file offset: %v", err)
	}
	return nil
}

// Close closes the WAL file.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.encoder != nil {
		if err := w.encoder.flush(); err != nil {
			return err
		}
	}
	return w.file.Close()
}

// Exist checks if the WAL directory exists
func Exist(dir string) bool {
	_, err := os.Stat(dir)
	return !os.IsNotExist(err)
}

// ValidSnapshotEntries returns valid snapshot entries
func ValidSnapshotEntries(logger *zap.Logger, dir string) ([]walpb.Snapshot, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.snap"))
	if err != nil {
		return nil, err
	}

	snapshots := make([]walpb.Snapshot, 0, len(files))
	for _, file := range files {
		var snap walpb.Snapshot
		data, err := os.ReadFile(file)
		if err != nil {
			logger.Warn("failed to read snapshot file", zap.String("path", file), zap.Error(err))
			continue
		}
		if err := json.Unmarshal(data, &snap); err != nil {
			logger.Warn("failed to unmarshal snapshot", zap.String("path", file), zap.Error(err))
			continue
		}
		snapshots = append(snapshots, snap)
	}
	return snapshots, nil
}

// Save implements the WAL save method
func (w *WAL) Save(st raftpb.HardState, ents []raftpb.Entry) error {
	// Serialize and save HardState
	if !raft.IsEmptyHardState(st) {
		data, err := st.Marshal()
		if err != nil {
			return err
		}
		if err := w.saveRecord(data); err != nil {
			return err
		}
	}

	// Save entries
	for _, ent := range ents {
		data, err := ent.Marshal()
		if err != nil {
			return err
		}
		if err := w.saveRecord(data); err != nil {
			return err
		}
	}
	return nil
}

// saveRecord saves a record to the WAL
func (w *WAL) saveRecord(data []byte) error {
	if w.file == nil {
		return errors.New("WAL file not opened")
	}
	// Write data length
	lenBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBuf, uint64(len(data)))
	if _, err := w.file.Write(lenBuf); err != nil {
		return err
	}
	// Write data
	if _, err := w.file.Write(data); err != nil {
		return err
	}
	return w.file.Sync()
}
