// Practice Booth - Master Lock (Hard)

const BOOTH_KEY = "lockpick_booth_2062";

function onEnter(user, room) {
    user.SetTempData(BOOTH_KEY, true);
    
    if (user.GetMiscData(BOOTH_KEY + "_completed")) {
        user.SendText("<ansi fg=\"yellow\">The Master Lock clicks open easily for you now.</ansi>");
    } else {
        user.SendText("<ansi fg=\"green\">The classic Master Lock. Escape for 30 gold!</ansi>");
    }
    return false;
}
