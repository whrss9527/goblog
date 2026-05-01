// music
//$(function () {
//    var mDiv = $('#music-163');
//    mDiv.mouseover(function () {
//        startMove(0);
//    });
//    mDiv.mouseout(function () {
//        startMove(-330);
//    });
//});
//var timer = null;
//
//function startMove(target) {
//    clearInterval(timer);
//    let mDiv = $('#music-163');
//    timer = setInterval(function () {
//        // 越靠近目标值速度越小
//        var speed = (target - mDiv.offset().left) / 20;
//        //向右移动速度为正数，向左为负数,将speed取整,显示不完全
//        speed = speed > 0 ? Math.ceil(speed) : Math.floor(speed);
//
//        if (mDiv.offset().left === target) {
//            clearInterval(timer);
//        } else {
//            mDiv.css('left', mDiv.offset().left + speed + "px")
//        }
//    }, 10);
//}



// document.write('<script src="{{.cdn}}/js/jquery.min.js"></script>')
// document.write('<script src="{{.cdn}}/js/bootstrap.min.js"></script>')
// document.write('<script src="{{.cdn}}/md/lib/marked.min.js"></script>')
// document.write('<script src="{{.cdn}}/md/lib/prettify.min.js"></script>')
//
// document.write('<script src="{{.cdn}}/md/lib/raphael.min.js"></script>')
// document.write('<script src="{{.cdn}}/md/lib/underscore.min.js"></script>')
// document.write('<script src="{{.cdn}}/md/lib/sequence-diagram.min.js"></script>')
// document.write('<script src="{{.cdn}}/md/lib/flowchart.min.js"></script>')
// document.write('<script src="{{.cdn}}/md/lib/jquery.flowchart.min.js"></script>')
// document.write('<script src="https://cdnjs.cloudflare.com/ajax/libs/moment.js/2.18.1/moment.min.js" charSet="utf-8"></script>')
// document.write('<script src="https://cdnjs.cloudflare.com/ajax/libs/d3/4.10.2/d3.min.js" charSet="utf-8"></script>')
// document.write('<script src="{{.cdn}}/heatmap/src/calendar-heatmap.js"></script>')
// document.write('<script src="{{.cdn}}/js/customization.js"></script>')
// document.write('<script src="{{.cdn}}/md/editormd.js"></script>')