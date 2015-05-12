// control conn<-----------
var CameraList = {
    Type : "CameraList",
    Rooms : {
        "room1_id" : {
            Name : "room1",
            SecretAddress : "room1-secret-addr",
            Cameras : {
                camera1_id : {
                    Name : "camera1",
                    Off : false,
                    Online : true
                },
                camera2_id : {
                    Name : "camera2",
                    Off : false,
                    Online : true
                }
            }
        }
    }
};

// signaling conn--------------->

// through path: /many/signaling/room1_id/camera1_id
var cameraId = "camnera1";

var OfferSignal = {
};

var IceSignal = {
};

// signaling conn<---------------

var AnswerSignal = {// standard answer
};

var RemoteIceSignal = {// standard ice
};