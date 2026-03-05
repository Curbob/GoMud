// Hacker Kit - CackalackyCon starter items

const ITEM_LIST = [
  10050, // lime juice squirter (weapon - alternative to schedule)
  20100, // cackalackycon tshirt (body armor)
  35001, // monster energy
  35001, // another monster energy
  35003, // hotel coffee
];

function onCommand_use(user, item, room) {
    
    SendUserMessage(user.UserId(), "You dig through the <ansi fg=\"itemname\">"+item.Name()+"</ansi> swag bag...");
    SendRoomMessage(room.RoomId(), user.GetCharacterName(true)+" digs through their <ansi fg=\"itemname\">"+item.Name()+"</ansi>, pulling out swag.", user.UserId());

    for ( var i=0; i<ITEM_LIST.length; i++) {
        item_id = ITEM_LIST[i];
        itm = CreateItem(item_id);
        SendUserMessage(user.UserId(), "You find a <ansi fg=\"itemname\">"+itm.Name()+"</ansi> inside!");
        user.GiveItem(itm);
    }

    SendUserMessage(user.UserId(), "");
    SendUserMessage(user.UserId(), "<ansi fg=\"yellow\">Tip: Get your badge from Base16 at Registration!</ansi>");

    item.AddUsesLeft(-1);
    item.MarkLastUsed();
 
    return true;
}
