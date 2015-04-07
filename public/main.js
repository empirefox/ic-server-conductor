var pc;
var ws;

function createOffer() {
	pc = RTCPeerConnection({
		constraints : {
			mandatory : {
				OfferToReceiveAudio : true,
				OfferToReceiveVideo : true
			},
			optional : []
		},

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
			console.log("Remote stream added:", URL.createObjectURL(event.stream));
			var remoteVideoElement = document.getElementById('remote-video');
			remoteVideoElement.src = URL.createObjectURL(event.stream);
			remoteVideoElement.play();
		},
		onRemoteStreamEnded : function(stream) {
			console.log('onRemoteStreamEnded');
		},

		onOfferSDP : function(offerSDP) {
			ws.send(JSON.stringify(offerSDP));
			trace("sent offer");
			console.log("sent offer:", offerSDP);
		}
	});
}

function startWs() {
	ws = new WebSocket('ws://127.0.0.1:9999/many');
	ws.onopen = function() {
		trace("ws opened");
		createOffer();
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
			case 'answer':
				pc.addAnswerSDP(signal);
				trace("get answer");
				break;
			case 'candidate':
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