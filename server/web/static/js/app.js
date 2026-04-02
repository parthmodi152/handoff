// Copy text to clipboard
function copyToClipboard(text, btn) {
    navigator.clipboard.writeText(text).then(function () {
        var original = btn.textContent;
        btn.textContent = "Copied!";
        setTimeout(function () {
            btn.textContent = original;
        }, 2000);
    });
}

// Auto-dismiss flash messages
document.addEventListener("DOMContentLoaded", function () {
    var flash = document.getElementById("flash-message");
    if (flash) {
        setTimeout(function () {
            flash.remove();
        }, 5000);
    }
});

// htmx after-swap: scroll to top on page navigation
document.addEventListener("htmx:afterSwap", function (event) {
    if (event.detail.target.id === "requests-table") {
        event.detail.target.scrollIntoView({ behavior: "smooth" });
    }
});
