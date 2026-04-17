package pixelforge_gui

import (
	"log"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pool"
)

func getPropagateToChildrenFromThePool() *propagateToChildren {
	propagate := propagateToChildrenPool.Get()
	propagateToChildrenToken++
	propagate.value = true
	propagate.token = propagateToChildrenToken
	return propagate
}

var propagateToChildrenToken = 0

// the object is valid only during element's Draw/Update.
type propagateToChildren struct {
	value bool
	token int // token is verified when setting the value.
}

func (p *propagateToChildren) set(v bool, token int) {
	if p.token == token { // when object is reused the token is changed
		p.value = v
	} else {
		log.Println("code was trying to stop event propagation after pigui element was updated/drawn")
	}
}

var propagateToChildrenPool pipool.Pool[propagateToChildren]
