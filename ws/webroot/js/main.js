define(
	"main",
	[
		"MessageList"
	],
	function(MessageList) {
		var ws = new WebSocket("ws://localhost:8080/ws");
		var list = new MessageList(ws);
		ko.applyBindings(list);
	}
);
