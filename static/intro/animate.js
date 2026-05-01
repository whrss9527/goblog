function loadingIt() {
    var $wrapper = $("#wrapper-placeholder").hide();
    var $loading_wrapper = $("#loading-placeholder").height($(window).height()).show();
    var $text = $("<p>").addClass("vertical").text("loading");
    var $tip = $("#loading-tip").addClass("animated wiggle");

    var fullWidth = $(window).width();
    var fullHeight = $(window).height();
    var fallCount = 25;
    var fallOrder = [];
    var fallIndex = 0;
    for (var i = 0; i < fallCount; i++) {
        fallOrder[i] = i;
    }
    for (i = 0; i < 100; i++) {
        var a = parseInt(Math.random() * (fallCount - 1));
        var b = parseInt(Math.random() * (fallCount - 1));
        var temp = fallOrder[a];
        fallOrder[a] = fallOrder[b];
        fallOrder[b] = temp;
    }

    function fallBlock() {
        i = fallOrder[fallIndex];
        var startPix = fullWidth * (i / (fallCount + 1));
        var endPix = fullWidth * ((i + 1) / (fallCount + 1));
        var rndPix = startPix + (endPix - startPix) * Math.random();
        var $block = $text.clone().css("left", rndPix);
        $loading_wrapper.append($block);
        $block.animate({ "top": fullHeight - 100, "opacity": 0 }, (0.5 + Math.random() / 2) * 3000);
        fallIndex++;
        if (fallIndex <= fallCount)
            setTimeout(fallBlock, (0.5 + Math.random() / 2) * 200);
        else
            flipIt();
    }

    function flipIt() {
        var $loaded = $("<p>").addClass("horizontal").text("加载完成...");
        $loading_wrapper.append($loaded);
        $loaded.addClass("animated lightSpeedIn");
        setTimeout(function () {
            var $copyLoaded = $loaded.clone();
            $($loading_wrapper).append($copyLoaded);
            $copyLoaded.addClass("animated lightSpeedIn");
        }, 30);
        $tip.fadeOut(300);
        setTimeout(normalIt, 1500);
    }

    function normalIt() {
        $loading_wrapper.fadeOut(1500);
        $wrapper.show();
        impress().init();
    }

    fallBlock();
}

loadingIt();