var Many_Raw_Message = {
    Content : "hello",
    Room : "null-all/room1_id",
    From : "user1-name",
    To : "null-all/user1-name"
};

var Many_Message = "many:chat:" + Many_Raw_Message;

var Many_RoomCommand = {
    Name : "SetRoomName",
    Room : "room1_id",
    RoomName : "new room name"
};

var Many_SetIpcamCommand = {
    Name : "SetIpcam",
    Room : "room1_id",
    Camera : "camera1_id",
    CameraName : "camera1",
    CameraOff : false
};

var Many_ReconnectIpcamCommand = {
    Name : "ReconnectIpcam",
    Room : "room1_id",
    Camera : "camera1_id"
};

var Command = "many:command:" + Message;