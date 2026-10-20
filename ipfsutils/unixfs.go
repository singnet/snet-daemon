package ipfsutils

import (
	"bytes"
	"context"
	"fmt"
	"math"

	"github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/boxo/ipld/unixfs"
	"github.com/ipfs/boxo/verifcid"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	mh "github.com/multiformats/go-multihash"
)

// Blocks must be validated as a unit. This bounds unverified response bodies,
// not the size of a file assembled from multiple blocks.
const maxBlockSize = 2 << 20

type blockFetcher func(context.Context, cid.Cid) ([]byte, error)

type unixFSFileNode struct {
	data  []byte
	links []*ipld.Link
	sizes []uint64
	size  uint64
}

func loadVerifiedFileNode(ctx context.Context, c cid.Cid, fetch blockFetcher) (*unixFSFileNode, error) {
	if err := verifcid.ValidateCid(verifcid.DefaultAllowlist, c); err != nil {
		return nil, fmt.Errorf("invalid block CID %s: %w", c, err)
	}
	if c.Type() != cid.Raw && c.Type() != cid.DagProtobuf {
		return nil, fmt.Errorf("unsupported codec 0x%x for block %s", c.Type(), c)
	}
	var data []byte
	var err error
	if c.Prefix().MhType == mh.IDENTITY {
		// Identity CIDs contain the block itself and need no network request.
		hash, decodeErr := mh.Decode(c.Hash())
		if decodeErr != nil {
			return nil, decodeErr
		}
		data = hash.Digest
	} else {
		data, err = fetch(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("fetch block %s: %w", c, err)
		}
	}
	if len(data) > maxBlockSize {
		return nil, fmt.Errorf("block %s exceeds the %d byte limit", c, maxBlockSize)
	}
	actual, err := c.Prefix().Sum(data)
	if err != nil {
		return nil, fmt.Errorf("hash block %s: %w", c, err)
	}
	if !actual.Equals(c) {
		return nil, fmt.Errorf("CID mismatch for block %s: received %s", c, actual)
	}
	if c.Type() == cid.Raw {
		return &unixFSFileNode{data: data, size: uint64(len(data))}, nil
	}
	// Decode only the original, verified bytes. Re-encoding a protobuf node can
	// change its CID even when it represents the same UnixFS data.
	node, err := merkledag.DecodeProtobuf(data)
	if err != nil {
		return nil, fmt.Errorf("decode DAG-PB block %s: %w", c, err)
	}
	fsNode, err := unixfs.FSNodeFromBytes(node.Data())
	if err != nil {
		return nil, fmt.Errorf("decode UnixFS block %s: %w", c, err)
	}
	if fsNode.Type() != unixfs.TFile && fsNode.Type() != unixfs.TRaw {
		return nil, fmt.Errorf("unsupported UnixFS type %s for block %s", fsNode.Type(), c)
	}
	links := node.Links()
	sizes := fsNode.BlockSizes()
	if len(links) != len(sizes) {
		return nil, fmt.Errorf("UnixFS block %s has inconsistent links and blocksizes", c)
	}
	total := uint64(len(fsNode.Data()))
	for _, size := range sizes {
		if size > math.MaxUint64-total {
			return nil, fmt.Errorf("UnixFS size overflow in block %s", c)
		}
		total += size
	}
	if total != fsNode.FileSize() {
		return nil, fmt.Errorf("UnixFS filesize mismatch in block %s", c)
	}
	return &unixFSFileNode{data: fsNode.Data(), links: links, sizes: sizes, size: total}, nil
}

func readVerifiedUnixFSFile(ctx context.Context, root cid.Cid, fetch blockFetcher) ([]byte, error) {
	type frame struct {
		cid   cid.Cid
		node  *unixFSFileNode
		next  int
		start int
	}
	var output bytes.Buffer
	stack := []frame{{cid: root}}
	active := make(map[cid.Cid]bool)
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		current := &stack[len(stack)-1]
		if current.node == nil {
			if active[current.cid] {
				return nil, fmt.Errorf("cycle in UnixFS file at block %s", current.cid)
			}
			node, err := loadVerifiedFileNode(ctx, current.cid, fetch)
			if err != nil {
				return nil, err
			}
			active[current.cid] = true
			current.node = node
			current.start = output.Len()
			_, _ = output.Write(node.data)
		}
		if current.next < len(current.node.links) {
			child := current.node.links[current.next].Cid
			current.next++
			stack = append(stack, frame{cid: child})
			continue
		}
		size := uint64(output.Len() - current.start)
		if size != current.node.size {
			return nil, fmt.Errorf("UnixFS filesize mismatch in block %s", current.cid)
		}
		delete(active, current.cid)
		stack = stack[:len(stack)-1]
		if len(stack) > 0 {
			parent := &stack[len(stack)-1]
			if size != parent.node.sizes[parent.next-1] {
				return nil, fmt.Errorf("UnixFS child size mismatch in block %s", parent.cid)
			}
		}
	}
	return output.Bytes(), nil
}
