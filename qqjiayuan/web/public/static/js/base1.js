//任务
function task() {
    var time = new Date();
    var utime = new Date(ntime.replace(/(-)/g, "/"));
    if (time > utime) {
        dpath = dpath.replace(/(%2e)/g, ".");
        dpath = dpath.replace(/(%2f)/g, "/");
        dpath = dpath.replace(/(%3f)/g, "?");
        dpath = dpath.replace(/(%3d)/g, "=");
        doAjax("" + dpath + "", "", "back", "GET", 0);
    }
}
function back(value) {
    if (value == "0") {
    }
}
//推送
function push() {
    (function () {
        if (location.host.indexOf(':443') > 0) {
            location.href = location.href.replace(':443', '');
            return false;
        }
        var bp = document.createElement('script');
        var curProtocol = window.location.protocol.split(':')[0];
        if (curProtocol === 'https') {
            bp.src = 'https://zz.bdstatic.com/linksubmit/push.js';
        }
        else {
            bp.src = 'http://push.zhanzhang.baidu.com/push.js';
        }
        var s = document.getElementsByTagName("script")[0];
        s.parentNode.insertBefore(bp, s);
    })();
}
//统计
function stat() {
    var _hmt = _hmt || [];
    (function () {
        var hm = document.createElement("script");
        hm.src = "https://hm.baidu.com/hm.js?db361b87ea42c3a0c050588c0131ff38";
        var s = document.getElementsByTagName("script")[0];
        s.parentNode.insertBefore(hm, s);
        document.writeln("<div style=\"display:none;\"><script src=\"https://s96.cnzz.com/z_stat.php?id=1000252883&web_id=1000252883\"></script></div>");
    })();
}
//链接
function link01() {
    //document.writeln("<script type=\"text/javascript\" src=\"/js.js\"></script>");
}
function link02() {
    //document.writeln("<script type=\"text/javascript\" src=\"/js.js\"></script>");
}
function link03() {
    //document.writeln("<script type=\"text/javascript\" src=\"/js.js\"></script>");
}