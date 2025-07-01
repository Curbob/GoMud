

function onPurchase(user, item, room) {

    var newRoomIds = CreateInstancesFromRoomIds( [432] );

    if ( newRoomIds[432] ) {

        SendUserMessage(user.UserId(), "You are directed to a room upstairs with a large bed. How inviting...");
        SendRoomMessage(room.RoomId(), user.GetCharacterName(true)+" says something to the Inn keeper and is escorted to another room.", user.UserId());
        
        user.MoveRoom(newRoomIds[432]);
        return false;
    } 

    return false;
}

