// Challenge Booth - Expert/Dark

const BOOTH_KEY = "lockpick_booth_2063";

function onEnter(user, room) {
    user.SetTempData(BOOTH_KEY, true);
    
    if (user.GetMiscData(BOOTH_KEY + "_completed")) {
        user.SendText("<ansi fg=\"red\">Darkness. But your fingers remember the way.</ansi>");
    } else {
        user.SendText("<ansi fg=\"red\">Complete darkness. 100 gold if you escape.</ansi>");
        user.SendText("<ansi fg=\"red\">Pick by feel alone.</ansi>");
    }
    return false;
}
