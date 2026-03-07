// Practice Booth - Brass Padlock (Medium)

const BOOTH_KEY = "lockpick_booth_2061";

function onEnter(user, room) {
    user.SetTempData(BOOTH_KEY, true);
    
    if (user.GetMiscData(BOOTH_KEY + "_completed")) {
        user.SendText("<ansi fg=\"yellow\">This brass padlock knows your touch now.</ansi>");
    } else {
        user.SendText("<ansi fg=\"green\">Security pins await. Escape for 15 gold!</ansi>");
    }
    return false;
}
