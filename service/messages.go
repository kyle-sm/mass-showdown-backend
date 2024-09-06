package service

type message struct {
	Type    messageType `json:"type"`
	Content interface{} `json:"content"`
}

type messageType string

const (
	poll            messageType = "NEW_POLL"
	updateRequest               = "UPDATE_REQ"
	updateResponse              = "UPDATE_RESP"
	vote                        = "VOTE"
	displayText                 = "DISPLAY_TEXT"
	showdownRequest             = "PS_REQUEST"
	results                     = "RESULTS"
	clearVote                   = "CLEAR_VOTE"
	wait                        = "WAIT"
	voteOk                      = "VOTE_OK"
)

type Vote struct {
	From string `json:"from,omitempty"`
	Type string `json:"type"`
	Idx  int    `json:"idx"`
	Tera bool   `json:"tera"`
}

type updateRequestMessage struct {
	From  string `json:"from"`
	Voted bool   `json:"voted"`
}

type updateResponseMessage struct {
	Results bool        `json:"results"`
	Update  interface{} `json:"update"`
}

type displayTextMessage struct {
	Clear   bool   `json:"clear"`
	Err     bool   `json:"err"`
	Message string `json:"message"`
}

type showdownRequestMessage struct {
	RoomID string
	Req    *PSBattleRequest
}

type pollResults struct {
	RoomID  string
	RQID    uint8
	Command string
}

type (
	PSBattleRequest struct {
		Wait        bool               `json:"wait"`
		ForceSwitch []bool             `json:"force_switch"`
		Active      []*PSActivePokemon `json:"active"`
		Side        PSSideInfo         `json:"side"`
		RQID        uint8              `json:"rqid"`
	}

	PSActivePokemon struct {
		Moves           []*PSMoveInfo `json:"moves"`
		CanTerastallize string        `json:"canTerastallize"`
		TeraVotes       float32       `json:"teraVotes,omitempty"`
	}

	PSMoveInfo struct {
		Move     string  `json:"move"`
		ID       string  `json:"id"`
		PP       uint8   `json:"pp"`
		MaxPP    uint8   `json:"maxpp"`
		Target   string  `json:"target"`
		Disabled bool    `json:"disabled"`
		Votes    float32 `json:"votes,omitempty"`
	}

	PSSideInfo struct {
		Name    string           `json:"name"`
		ID      string           `json:"id"`
		Pokemon []*PSSidePokemon `json:"pokemon"`
	}

	PSSidePokemon struct {
		Ident         string            `json:"ident"`
		Details       string            `json:"details"`
		Condition     string            `json:"condition"`
		Active        bool              `json:"active"`
		Stats         map[string]uint16 `json:"stats"`
		Moves         []string          `json:"moves"`
		BaseAbility   string            `json:"base_ability"`
		Item          string            `json:"item"`
		Pokeball      string            `json:"pokeball"`
		Ability       string            `json:"ability"`
		Commanding    bool              `json:"commanding"`
		Reviving      bool              `json:"reviving"`
		TeraType      string            `json:"teraType"`
		Terastallized string            `json:"terastallized"`
		Votes         float32           `json:"votes,omitempty"`
	}
)
