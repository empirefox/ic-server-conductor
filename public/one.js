var pc;
var ws;
var stream;

var getUserMedia = (navigator.getUserMedia || navigator.webkitGetUserMedia || navigator.mozGetUserMedia || navigator.msGetUserMedia).bind(navigator);

function createAnswer(offerSDP_sent_from_offerer) {
	pc = RTCPeerConnection({
		constraints : {
			mandatory : {
				OfferToReceiveAudio : false,
				OfferToReceiveVideo : false
			},
			optional : []
		},

		attachStream : stream,

		onICE : function(candidate) {
			console.log("onicecandidate:", candidate);
			candidate = {
                type : "candidate",
				sdpMLineIndex : candidate.sdpMLineIndex,
				candidate : candidate.candidate
			};
			sendToPeer(JSON.stringify(candidate));
		},
		onRemoteStream : function(stream) {
			console.log('onRemoteStream: should not be called');
		},
		onRemoteStreamEnded : function(stream) {
			console.log('onRemoteStreamEnded: should not be called');
		},

		offerSDP : offerSDP_sent_from_offerer,

		onAnswerSDP : function(answerSDP) {
			ws.send(JSON.stringify(answerSDP));
			trace("sent answer");
			console.log("sent answer:", answerSDP);
		}
	});
}

function startWs() {
	ws = new WebSocket('ws://127.0.0.1:9999/one');
	ws.onopen = function() {
		trace("ws opened");
		getUserMedia({
			"audio" : true,
			"video" : true
		}, function(s) {
			trace("media got");
			stream = s;
			var videoElement = document.getElementById('video');
			videoElement.src = URL.createObjectURL(s);
			videoElement.play();
		}, function(e) {
			trace('getUserMedia failed: ' + JSON.stringify(e, null, '\t'));
		});
	};
	ws.onclose = function() {
		trace("ws closed");
		document.getElementById("connect").disabled = false;
		document.getElementById("disconnect").disabled = true;
	};
	ws.onmessage = function(e) {
		trace("ws.onmessage");
		console.log('ws.onmessage:', e.data);
		var signal = JSON.parse(e.data);
		switch(signal.type) {
			case 'offer':
				createAnswer(signal);
				break;
			case 'candidate':
				if (!pc) {
					console.log('ws.onmessage.candidate:no pc');
					return;
				}
                pc.addICE(signal);
				trace("get candidate");
		}
	};
}

function trace(txt) {
	var elem = document.getElementById("debug");
	elem.innerHTML += txt + "<br>";
}

function sendToPeer(data) {
	console.log("Send singal", data);
	ws.send(data);
}

function connect() {
	localName = document.getElementById("local").value.toLowerCase();
	server = document.getElementById("server").value.toLowerCase();
	if (localName.length == 0) {
		alert("I need a name please.");
		document.getElementById("local").focus();
	} else {
		document.getElementById("connect").disabled = true;
		document.getElementById("disconnect").disabled = false;
		startWs();
	}
}

function disconnect() {
	if (ws.readyState < 2) {
		ws.close();
	}
}

window.onbeforeunload = disconnect;

function toggleMe(obj) {
	var id = obj.id.replace("toggle", "msg");
	var t = document.getElementById(id);
	if (obj.innerText == "+") {
		obj.innerText = "-";
		t.style.display = "block";
	} else {
		obj.innerText = "+";
		t.style.display = "none";
	}
}