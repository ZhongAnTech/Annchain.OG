// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package fetcher contains the sequencer announcement based synchronisation.
package fetcher

import (
	"errors"
	"math/rand"
	"time"

	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

const (
	arriveTimeout  = 500 * time.Millisecond // Time allowance before an announced sequencer is explicitly requested
	gatherSlack    = 100 * time.Millisecond // Interval used to collate almost-expired announces with fetches
	fetchTimeout   = 5 * time.Second        // Maximum allotted time to return an explicitly requested sequencer
	maxUncleDist   = 7                      // Maximum allowed backward distance from the chain head
	maxQueueDist   = 32                     // Maximum allowed distance from the chain head to queue
	hashLimit      = 256                    // Maximum number of unique sequencers a peer may have announced
	sequencerLimit = 64                     // Maximum number of unique sequencers a peer may have delivered
)

var (
	errTerminated = errors.New("terminated")
)

// sequencerRetrievalFn is a callback type for retrieving a sequencer from the local chain.
type sequencerRetrievalFn func(types.Hash) *types.Sequencer

// headerRequesterFn is a callback type for sending a header retrieval request.
type headerRequesterFn func(types.Hash) error

// bodyRequesterFn is a callback type for sending a body retrieval request.
type bodyRequesterFn func([]types.Hash) error

// chainHeightFn is a callback type to retrieve the current chain height.
type chainHeightFn func() uint64

// chainInsertFn is a callback type to insert a batch of sequencers into the local chain.
type chainInsertFn func(seq *types.Sequencer, txs types.Txs) error

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

// announce is the hash notification of the availability of a new sequencer in the
// network.
type announce struct {
	hash   types.Hash             // Hash of the sequencer being announced
	number uint64                 // Number of the sequencer being announced (0 = unknown | old protocol)
	header *types.SequencerHeader // Header of the sequencer partially reassembled (new protocol)
	time   time.Time              // Timestamp of the announcement

	origin string // Identifier of the peer originating the notification

	fetchHeader headerRequesterFn // Fetcher function to retrieve the header of an announced sequencer
	fetchBodies bodyRequesterFn   // Fetcher function to retrieve the body of an announced sequencer
}

// headerFilterTask represents a batch of headers needing fetcher filtering.
type headerFilterTask struct {
	peer    string                   // The source peer of sequencer headers
	headers []*types.SequencerHeader // Collection of headers to filter
	time    time.Time                // Arrival time of the headers
}

// bodyFilterTask represents a batch of sequencer bodies (transactions and uncles)
// needing fetcher filtering.
type bodyFilterTask struct {
	peer         string        // The source peer of sequencer bodies
	transactions [][]*types.Tx // Collection of transactions per sequencer bodies
	sequencers   []*types.Sequencer
	time         time.Time // Arrival time of the sequencers' contents
}

// inject represents a schedules import operation.
type inject struct {
	origin    string
	sequencer *types.Sequencer
}

// Fetcher is responsible for accumulating sequencer announcements from various peers
// and scheduling them for retrieval.
type Fetcher struct {
	// Various event channels
	notify chan *announce
	inject chan *inject

	sequencerFilter chan chan []*types.Sequencer
	headerFilter    chan chan *headerFilterTask
	bodyFilter      chan chan *bodyFilterTask

	done chan types.Hash
	quit chan struct{}

	// Announce states
	announces  map[string]int             // Per peer announce counts to prevent memory exhaustion
	announced  map[types.Hash][]*announce // Announced sequencers, scheduled for fetching
	fetching   map[types.Hash]*announce   // Announced sequencers, currently fetching
	fetched    map[types.Hash][]*announce // sequencers with headers fetched, scheduled for body retrieval
	completing map[types.Hash]*announce   // sequencers with headers, currently body-completing

	// sequencer cache
	queue  *prque.Prque           // Queue containing the import operations (sequencer number sorted)
	queues map[string]int         // Per peer sequencer counts to prevent memory exhaustion
	queued map[types.Hash]*inject // Set of already queued sequencers (to dedupe imports)

	getsequencer sequencerRetrievalFn

	chainHeight chainHeightFn // Retrieves the current chain's height
	insertChain chainInsertFn // Injects a batch of sequencers into the chain
	dropPeer    peerDropFn    // Drops a peer for misbehaving

	// Testing hooks
	announceChangeHook func(types.Hash, bool)           // Method to call upon adding or deleting a hash from the announce list
	queueChangeHook    func(types.Hash, bool)           // Method to call upon adding or deleting a sequencer from the import queue
	fetchingHook       func([]types.Hash)               // Method to call upon starting a sequencer (eth/61) or header (eth/62) fetch
	completingHook     func([]types.Hash)               // Method to call upon starting a sequencer body fetch (eth/62)
	importedHook       func(sequencer *types.Sequencer) // Method to call upon successful sequencer import (both eth/61 and eth/62)
}

// New creates a sequencer fetcher to retrieve sequencers based on hash announcements.
func New(getsequencer sequencerRetrievalFn, chainHeight chainHeightFn, insertChain chainInsertFn, dropPeer peerDropFn) *Fetcher {
	return &Fetcher{
		notify:          make(chan *announce),
		inject:          make(chan *inject),
		sequencerFilter: make(chan chan []*types.Sequencer),
		headerFilter:    make(chan chan *headerFilterTask),
		bodyFilter:      make(chan chan *bodyFilterTask),
		done:            make(chan types.Hash),
		quit:            make(chan struct{}),
		announces:       make(map[string]int),
		announced:       make(map[types.Hash][]*announce),
		fetching:        make(map[types.Hash]*announce),
		fetched:         make(map[types.Hash][]*announce),
		completing:      make(map[types.Hash]*announce),
		queue:           prque.New(),
		queues:          make(map[string]int),
		queued:          make(map[types.Hash]*inject),
		getsequencer:    getsequencer,
		chainHeight:     chainHeight,
		insertChain:     insertChain,
		dropPeer:        dropPeer,
	}
}

// Start boots up the announcement based synchroniser, accepting and processing
// hash notifications and sequencer fetches until termination requested.
func (f *Fetcher) Start() {
	go f.loop()
}

// Stop terminates the announcement based synchroniser, canceling all pending
// operations.
func (f *Fetcher) Stop() {
	close(f.quit)
}

// Notify announces the fetcher of the potential availability of a new sequencer in
// the network.
func (f *Fetcher) Notify(peer string, hash types.Hash, number uint64, time time.Time,
	headerFetcher headerRequesterFn, bodyFetcher bodyRequesterFn) error {
	sequencer := &announce{
		hash:        hash,
		number:      number,
		time:        time,
		origin:      peer,
		fetchHeader: headerFetcher,
		fetchBodies: bodyFetcher,
	}
	select {
	case f.notify <- sequencer:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Enqueue tries to fill gaps the the fetcher's future import queue.
func (f *Fetcher) Enqueue(peer string, sequencer *types.Sequencer) error {
	op := &inject{
		origin:    peer,
		sequencer: sequencer,
	}
	select {
	case f.inject <- op:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// FilterHeaders extracts all the headers that were explicitly requested by the fetcher,
// returning those that should be handled differently.
func (f *Fetcher) FilterHeaders(peer string, headers []*types.SequencerHeader, time time.Time) []*types.SequencerHeader {
	log.WithField("peer", peer).WithField("headers", len(headers)).Debug("Filtering headers")

	// Send the filter channel to the fetcher
	filter := make(chan *headerFilterTask)

	select {
	case f.headerFilter <- filter:
	case <-f.quit:
		return nil
	}
	// Request the filtering of the header list
	select {
	case filter <- &headerFilterTask{peer: peer, headers: headers, time: time}:
	case <-f.quit:
		return nil
	}
	// Retrieve the headers remaining after filtering
	select {
	case task := <-filter:
		return task.headers
	case <-f.quit:
		return nil
	}
}

// FilterBodies extracts all the sequencer bodies that were explicitly requested by
// the fetcher, returning those that should be handled differently.
func (f *Fetcher) FilterBodies(peer string, transactions [][]*types.Tx, sequencers []*types.Sequencer, time time.Time) [][]*types.Tx {
	log.WithField("txs", len(transactions)).WithField("sequencers ", len(sequencers)).WithField("peer", peer).Debug("Filtering bodies")

	// Send the filter channel to the fetcher
	filter := make(chan *bodyFilterTask)

	select {
	case f.bodyFilter <- filter:
	case <-f.quit:
		return nil
	}
	// Request the filtering of the body list
	select {
	case filter <- &bodyFilterTask{peer: peer, transactions: transactions, sequencers: sequencers, time: time}:
	case <-f.quit:
		return nil
	}
	// Retrieve the bodies remaining after filtering
	select {
	case task := <-filter:
		return task.transactions
	case <-f.quit:
		return nil
	}
}

// Loop is the main fetcher loop, checking and processing various notification
// events.
func (f *Fetcher) loop() {
	// Iterate the sequencer fetching until a quit is requested
	fetchTimer := time.NewTimer(0)
	completeTimer := time.NewTimer(0)

	for {
		// Clean up any expired sequencer fetches
		for hash, announce := range f.fetching {
			if time.Since(announce.time) > fetchTimeout {
				f.forgetHash(hash)
			}
		}
		// Import any queued sequencers that could potentially fit
		height := f.chainHeight()
		for !f.queue.Empty() {
			op := f.queue.PopItem().(*inject)
			hash := op.sequencer.GetTxHash()
			if f.queueChangeHook != nil {
				f.queueChangeHook(hash, false)
			}
			// If too high up the chain or phase, continue later
			number := op.sequencer.Id
			if number > height+1 {
				f.queue.Push(op, -float32(number))
				if f.queueChangeHook != nil {
					f.queueChangeHook(hash, true)
				}
				break
			}
			// Otherwise if fresh and still unknown, try and import
			if number+maxUncleDist < height || f.getsequencer(hash) != nil {
				f.forgetsequencer(hash)
				continue
			}
			//todo
			f.insert(op.origin, op.sequencer, nil)
		}
		// Wait for an outside event to occur
		select {
		case <-f.quit:
			// Fetcher terminating, abort all operations
			return

		case notification := <-f.notify:
			// A sequencer was announced, make sure the peer isn't DOSing us
			propAnnounceInMeter.Mark(1)

			count := f.announces[notification.origin] + 1
			if count > hashLimit {
				log.WithField("peer", notification.origin).WithField(
					"limit", hashLimit).Debug("Peer exceeded outstanding announces")
				propAnnounceDOSMeter.Mark(1)
				break
			}
			// If we have a valid sequencer number, check that it's potentially useful
			if notification.number > 0 {
				if dist := int64(notification.number) - int64(f.chainHeight()); dist < -maxUncleDist || dist > maxQueueDist {
					log.WithField("peer", notification.origin).WithField("number", notification.number).WithField(
						"hash", notification.hash).WithField("distance", dist).Debug("Peer discarded announcement")
					propAnnounceDropMeter.Mark(1)
					break
				}
			}
			// All is well, schedule the announce if sequencer's not yet downloading
			if _, ok := f.fetching[notification.hash]; ok {
				break
			}
			if _, ok := f.completing[notification.hash]; ok {
				break
			}
			f.announces[notification.origin] = count
			f.announced[notification.hash] = append(f.announced[notification.hash], notification)
			if f.announceChangeHook != nil && len(f.announced[notification.hash]) == 1 {
				f.announceChangeHook(notification.hash, true)
			}
			if len(f.announced) == 1 {
				f.rescheduleFetch(fetchTimer)
			}

		case op := <-f.inject:
			// A direct sequencer insertion was requested, try and fill any pending gaps
			propBroadcastInMeter.Mark(1)
			f.enqueue(op.origin, op.sequencer)

		case hash := <-f.done:
			// A pending import finished, remove all traces of the notification
			f.forgetHash(hash)
			f.forgetsequencer(hash)

		case <-fetchTimer.C:
			// At least one sequencer's timer ran out, check for needing retrieval
			request := make(map[string][]types.Hash)

			for hash, announces := range f.announced {
				if time.Since(announces[0].time) > arriveTimeout-gatherSlack {
					// Pick a random peer to retrieve from, reset all others
					announce := announces[rand.Intn(len(announces))]
					f.forgetHash(hash)

					// If the sequencer still didn't arrive, queue for fetching
					if f.getsequencer(hash) == nil {
						request[announce.origin] = append(request[announce.origin], hash)
						f.fetching[hash] = announce
					}
				}
			}
			// Send out all sequencer header requests
			for peer, hashes := range request {
				log.Debug("Fetching scheduled headers", "peer", peer, "list", hashes)

				// Create a closure of the fetch and schedule in on a new thread
				fetchHeader, hashes := f.fetching[hashes[0]].fetchHeader, hashes
				go func() {
					if f.fetchingHook != nil {
						f.fetchingHook(hashes)
					}
					for _, hash := range hashes {
						headerFetchMeter.Mark(1)
						fetchHeader(hash) // Suboptimal, but protocol doesn't allow batch header retrievals
					}
				}()
			}
			// Schedule the next fetch if sequencers are still pending
			f.rescheduleFetch(fetchTimer)

		case <-completeTimer.C:
			// At least one header's timer ran out, retrieve everything
			request := make(map[string][]types.Hash)

			for hash, announces := range f.fetched {
				// Pick a random peer to retrieve from, reset all others
				announce := announces[rand.Intn(len(announces))]
				f.forgetHash(hash)

				// If the sequencer still didn't arrive, queue for completion
				if f.getsequencer(hash) == nil {
					request[announce.origin] = append(request[announce.origin], hash)
					f.completing[hash] = announce
				}
			}
			// Send out all sequencer body requests
			for peer, hashes := range request {
				log.Debug("Fetching scheduled bodies", "peer", peer, "list", hashes)

				// Create a closure of the fetch and schedule in on a new thread
				if f.completingHook != nil {
					f.completingHook(hashes)
				}
				bodyFetchMeter.Mark(int64(len(hashes)))
				go f.completing[hashes[0]].fetchBodies(hashes)
			}
			// Schedule the next fetch if sequencers are still pending
			f.rescheduleComplete(completeTimer)

		case filter := <-f.headerFilter:
			// Headers arrived from a remote peer. Extract those that were explicitly
			// requested by the fetcher, and return everything else so it's delivered
			// to other parts of the system.
			var task *headerFilterTask
			select {
			case task = <-filter:
			case <-f.quit:
				return
			}
			headerFilterInMeter.Mark(int64(len(task.headers)))

			// Split the batch of headers into unknown ones (to return to the caller),
			// known incomplete ones (requiring body retrievals) and completed sequencers.
			unknown, incomplete, complete := []*types.SequencerHeader{}, []*announce{}, []*types.Sequencer{}
			for _, header := range task.headers {
				hash := header.Hash()

				// Filter fetcher-requested headers from other synchronisation algorithms
				if announce := f.fetching[hash]; announce != nil && announce.origin == task.peer && f.fetched[hash] == nil && f.completing[hash] == nil && f.queued[hash] == nil {
					// If the delivered header does not match the promised number, drop the announcer
					if header.SequencerId() != announce.number {
						log.WithField("peer", announce.origin).WithField("hash", header.Hash()).WithField("announced", announce.number).WithField(
							"provided", header.SequencerId()).Debug("Invalid sequencer number fetched")
						f.dropPeer(announce.origin)
						f.forgetHash(hash)
						continue
					}
					// Only keep if not imported by other means
					if f.getsequencer(hash) == nil {
						announce.header = header
						announce.time = task.time

						// Otherwise add to the list of sequencers needing completion
						incomplete = append(incomplete, announce)
					} else {
						log.WithField("peer", announce.origin).WithField("hash", header.Hash()).WithField(
							"number", header.SequencerId()).Debug("quencer already imported, discarding header")

						f.forgetHash(hash)
					}
				} else {
					// Fetcher doesn't know about it, add to the return list
					unknown = append(unknown, header)
				}
			}
			headerFilterOutMeter.Mark(int64(len(unknown)))
			select {
			case filter <- &headerFilterTask{headers: unknown, time: task.time}:
			case <-f.quit:
				return
			}
			// Schedule the retrieved headers for body completion
			for _, announce := range incomplete {
				hash := announce.header.Hash()
				if _, ok := f.completing[hash]; ok {
					continue
				}
				f.fetched[hash] = append(f.fetched[hash], announce)
				if len(f.fetched) == 1 {
					f.rescheduleComplete(completeTimer)
				}
			}
			// Schedule the header-only sequencers for import
			for _, sequencer := range complete {
				if announce := f.completing[sequencer.GetTxHash()]; announce != nil {
					f.enqueue(announce.origin, sequencer)
				}
			}

		case filter := <-f.bodyFilter:
			// sequencer bodies arrived, extract any explicitly requested sequencers, return the rest
			var task *bodyFilterTask
			select {
			case task = <-filter:
			case <-f.quit:
				return
			}
			bodyFilterInMeter.Mark(int64(len(task.transactions)))
			sequencers := []*types.Sequencer{}
			bodyFilterOutMeter.Mark(int64(len(task.transactions)))
			select {
			case filter <- task:
			case <-f.quit:
				return
			}
			// Schedule the retrieved sequencers for ordered import
			for _, sequencer := range sequencers {
				if announce := f.completing[sequencer.GetTxHash()]; announce != nil {
					f.enqueue(announce.origin, sequencer)
				}
			}
		}
	}
}

// rescheduleFetch resets the specified fetch timer to the next announce timeout.
func (f *Fetcher) rescheduleFetch(fetch *time.Timer) {
	// Short circuit if no sequencers are announced
	if len(f.announced) == 0 {
		return
	}
	// Otherwise find the earliest expiring announcement
	earliest := time.Now()
	for _, announces := range f.announced {
		if earliest.After(announces[0].time) {
			earliest = announces[0].time
		}
	}
	fetch.Reset(arriveTimeout - time.Since(earliest))
}

// rescheduleComplete resets the specified completion timer to the next fetch timeout.
func (f *Fetcher) rescheduleComplete(complete *time.Timer) {
	// Short circuit if no headers are fetched
	if len(f.fetched) == 0 {
		return
	}
	// Otherwise find the earliest expiring announcement
	earliest := time.Now()
	for _, announces := range f.fetched {
		if earliest.After(announces[0].time) {
			earliest = announces[0].time
		}
	}
	complete.Reset(gatherSlack - time.Since(earliest))
}

// enqueue schedules a new future import operation, if the sequencer to be imported
// has not yet been seen.
func (f *Fetcher) enqueue(peer string, sequencer *types.Sequencer) {
	hash := sequencer.GetTxHash()

	// Ensure the peer isn't DOSing us
	count := f.queues[peer] + 1
	if count > sequencerLimit {
		log.WithField("peer", peer).WithField("number", sequencer.Number()).WithField("hash", hash).WithField(
			"limit", sequencerLimit).Debug("Discarded propagated sequencer, exceeded allowance")
		propBroadcastDOSMeter.Mark(1)
		f.forgetHash(hash)
		return
	}
	// Discard any past or too distant sequencers
	if dist := int64(sequencer.Number()) - int64(f.chainHeight()); dist < -maxUncleDist || dist > maxQueueDist {
		log.WithField("peer", peer).WithField("number", sequencer.Number()).WithField("hash", hash).WithField("distance", dist)
		propBroadcastDropMeter.Mark(1)
		f.forgetHash(hash)
		return
	}
	// Schedule the sequencer for future importing
	if _, ok := f.queued[hash]; !ok {
		op := &inject{
			origin:    peer,
			sequencer: sequencer,
		}
		f.queues[peer] = count
		f.queued[hash] = op
		f.queue.Push(op, -float32(sequencer.Number()))
		if f.queueChangeHook != nil {
			f.queueChangeHook(op.sequencer.GetTxHash(), true)
		}
		log.WithField("peer", peer).WithField("Number", sequencer.Number()).WithField("hash", hash).WithField(
			"queued", f.queue.Size()).Debug("Queued propagated sequencer")
	}
}

// insert spawns a new goroutine to run a sequencer insertion into the chain. If the
// sequencer's number is at the same height as the current import phase, it updates
// the phase states accordingly.
func (f *Fetcher) insert(peer string, sequencer *types.Sequencer, txs []*types.Tx) {
	hash := sequencer.GetTxHash()

	// Run the import on a new thread
	log.WithField("peer", peer).WithField("number", sequencer.Number()).WithField(
		"hash", hash).Debug("Importing propagated sequencer")
	go func() {
		defer func() { f.done <- hash }()

		// Run the actual import and log any issues
		if err := f.insertChain(sequencer, txs); err != nil {
			log.WithField("peer", peer).WithField("number", sequencer.Number()).WithField(
				"hash", hash).WithError(err).Debug("Propagated sequencer import failed")
			return
		}

		// Invoke the testing hook if needed
		if f.importedHook != nil {
			f.importedHook(sequencer)
		}
	}()
}

// forgetHash removes all traces of a sequencer announcement from the fetcher's
// internal state.
func (f *Fetcher) forgetHash(hash types.Hash) {
	// Remove all pending announces and decrement DOS counters
	for _, announce := range f.announced[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.announced, hash)
	if f.announceChangeHook != nil {
		f.announceChangeHook(hash, false)
	}
	// Remove any pending fetches and decrement the DOS counters
	if announce := f.fetching[hash]; announce != nil {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
		delete(f.fetching, hash)
	}

	// Remove any pending completion requests and decrement the DOS counters
	for _, announce := range f.fetched[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.fetched, hash)

	// Remove any pending completions and decrement the DOS counters
	if announce := f.completing[hash]; announce != nil {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
		delete(f.completing, hash)
	}
}

// forgetsequencer removes all traces of a queued sequencer from the fetcher's internal
// state.
func (f *Fetcher) forgetsequencer(hash types.Hash) {
	if insert := f.queued[hash]; insert != nil {
		f.queues[insert.origin]--
		if f.queues[insert.origin] == 0 {
			delete(f.queues, insert.origin)
		}
		delete(f.queued, hash)
	}
}
