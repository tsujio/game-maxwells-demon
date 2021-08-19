package touchutil

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type TouchContext struct {
	isTouchIDsJustStored bool
	touchIDs             []ebiten.TouchID
}

func CreateTouchContext() *TouchContext {
	return &TouchContext{}
}

func (c *TouchContext) Update() {
	if touchIDs := inpututil.JustPressedTouchIDs(); len(touchIDs) > 0 {
		c.touchIDs = touchIDs
		c.isTouchIDsJustStored = true
	} else {
		c.isTouchIDsJustStored = false
	}
}

func (c *TouchContext) IsJustTouched() bool {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	if c.isTouchIDsJustStored {
		return true
	}
	return false
}

func (c *TouchContext) IsJustReleased() bool {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		return true
	}
	for _, id := range c.touchIDs {
		if inpututil.IsTouchJustReleased(id) {
			return true
		}
	}
	return false
}
