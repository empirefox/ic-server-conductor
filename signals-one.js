// control conn-------->
var IpcamsInfo_base = {
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
};
var IpcamsInfo = "one:IpcamsInfo:" + IpcamsInfo_base;

var One_Raw_Log = {
    Level : "info",
    Content : "ipcam camera1 come online"
};
var OneLog = "one:OneLog:" + One_Raw_Log;

// control conn<---------
var GetIpcamsInfo = {
    Name : "GetIpcamsInfo"
};

// from path: /many/signaling/room1_id/camera1_id
var CreateSignalingConnection = {
    Name : "CreateSignalingConnection",
    Reciever : "id_of_many_client",
    Camera : camera1_id
};

var ForceReRegistry = {
    Name : "ForceReRegistry",
    Camera : camera1_id
};
