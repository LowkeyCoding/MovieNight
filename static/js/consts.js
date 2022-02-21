const ColorRegex = /^([0-9A-Fa-f]{3}){1,2}$/;

const ClientDataType = {
    CdMessage: 0,
    CdUsers: 1,
    CdPing: 2,
    CdAuth: 3,
    CdColor: 4,
    CdEmote: 5,
    CdJoin: 6,
    CdNotify: 7,
};
Object.freeze(ClientDataType);

const DataType = {
	DTInvalid: 0,
	DTChat: 1,
	DTCommand: 2,
	DTEvent: 3,
	DTClient: 4,
	DTHidden: 5,
};
Object.freeze(DataType);

const CommandType = {
    CmdPlaying: 0,
    CmdRefreshPlayer: 1,
    CmdPurgeChat: 2,
    CmdHelp: 3,
    CmdEmotes: 4,
};
Object.freeze(CommandType);

const EventType = {
    EvJoin: 0,
    EvLeave: 1,
    EvKick: 2,
    EvBan: 3,
    EvServerMessage: 4,
    EvNameChange: 5,
    EvNameChangeForced: 6,
};
Object.freeze(EventType);

const MessageType = {
    MsgChat: 0,
    MsgAction: 1,
    MsgServer: 2,
    MsgError: 3,
    MsgNotice: 4,
    MsgCommandResponse: 5,
    MsgCommandError: 6,
};
Object.freeze(MessageType);

const Colors = [
    "aliceblue",
    "antiquewhite",
    "aqua",
    "aquamarine",
    "azure",
    "beige",
    "bisque",
    "blanchedalmond",
    "burlywood",
    "cadetblue",
    "chartreuse",
    "chocolate",
    "coral",
    "cornflowerblue",
    "cornsilk",
    "cyan",
    "darkcyan",
    "darkgoldenrod",
    "darkgray",
    "darkkhaki",
    "darkorange",
    "darksalmon",
    "darkseagreen",
    "darkturquoise",
    "deeppink",
    "deepskyblue",
    "dodgerblue",
    "floralwhite",
    "fuchsia",
    "gainsboro",
    "ghostwhite",
    "gold",
    "goldenrod",
    "gray",
    "greenyellow",
    "honeydew",
    "hotpink",
    "ivory",
    "khaki",
    "lavender",
    "lavenderblush",
    "lawngreen",
    "lemonchiffon",
    "lightblue",
    "lightcoral",
    "lightcyan",
    "lightgoldenrodyellow",
    "lightgreen",
    "lightgrey",
    "lightpink",
    "lightsalmon",
    "lightseagreen",
    "lightskyblue",
    "lightslategray",
    "lightsteelblue",
    "lightyellow",
    "lime",
    "limegreen",
    "linen",
    "magenta",
    "mediumaquamarine",
    "mediumorchid",
    "mediumpurple",
    "mediumseagreen",
    "mediumslateblue",
    "mediumspringgreen",
    "mediumturquoise",
    "mintcream",
    "mistyrose",
    "moccasin",
    "navajowhite",
    "oldlace",
    "olive",
    "olivedrab",
    "orange",
    "orangered",
    "orchid",
    "palegoldenrod",
    "palegreen",
    "paleturquoise",
    "palevioletred",
    "papayawhip",
    "peachpuff",
    "peru",
    "pink",
    "plum",
    "powderblue",
    "red",
    "rosybrown",
    "salmon",
    "sandybrown",
    "seagreen",
    "seashell",
    "silver",
    "skyblue",
    "slategray",
    "snow",
    "springgreen",
    "steelblue",
    "tan",
    "thistle",
    "tomato",
    "turquoise",
    "violet",
    "wheat",
    "white",
    "whitesmoke",
    "yellow",
    "yellowgreen",
]
Object.freeze(Colors);
