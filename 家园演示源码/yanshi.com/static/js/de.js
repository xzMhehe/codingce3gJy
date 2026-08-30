

 function vvbb()
{ 
var strhtml ="<div id='SmohanFaceBox6'><div class='Content'><h3><span>引用ubb</span><a class='close' href='javascript:hiddenface6(1);' title='关闭'></a></h3><ul><img src='/xml/Template/default/img/loads.gif' alt='.'/>拼命加载中...</ul></div></div>" ;
document.getElementById('showfhg').innerHTML = strhtml; myajax('GET','/xml/bbs/ajax.aspx?aa=6','bbsface6');
}  
function hiddenface6(a) { if(a==1){
document.getElementById('SmohanFaceBox6').style.display = 'none'; 
}else {
document.getElementById('SmohanFaceBox6').style.display = 'block';
document.getElementById('SmohanFaceBox').style.display = 'none'; 
document.getElementById('SmohanFaceBoxa').style.display = 'none'; 
document.getElementById('SmohanFaceBoxb').style.display = 'none';
document.getElementById('SmohanFaceBoxc').style.display = 'none';
document.getElementById('SmohanFaceBoxd').style.display = 'none';
} }

 function showface88()
{ 
 var strhtml ="<div id='SmohanFaceBox'><span class='Corner'></span><div class='Content'><h3><span>常用表情</span><a class='close' href='javascript:hiddenface(1);' title='关闭'></a></h3><ul><img src='/xml/Template/default/img/loads.gif' alt='.'/>正在加载中...</ul></div></div>" ;
document.getElementById('showfacelist').innerHTML = strhtml; myajax('GET','/wml/bbs/ajax.aspx?','bbsface');
}  


function showface()
{ 
 var strhtml ="<div id='SmohanFaceBox'><span class='Corner'></span><div class='Content'><h3><span>常用表情</span><a class='close' href='javascript:hiddenface(1);' title='关闭'></a></h3><ul><img src='/xml/Template/default/img/loads.gif' alt='.'/>正在加载中...</ul></div></div>" ;
document.getElementById('showfacelist').innerHTML = strhtml; myajax('GET','/wml/bbs/ajax.aspx?','bbsface');
}  
function hiddenface(a) { if(a==1){
document.getElementById('SmohanFaceBox').style.display = 'none'; 
}else {
document.getElementById('SmohanFaceBox').style.display = 'block';
document.getElementById('SmohanFaceBox6').style.display = 'none'; 
document.getElementById('SmohanFaceBoxa').style.display = 'none'; 
document.getElementById('SmohanFaceBoxb').style.display = 'none';
document.getElementById('SmohanFaceBoxc').style.display = 'none';
document.getElementById('SmohanFaceBoxd').style.display = 'none';
} }
 function addubb(ubb) {
var textEl = f.content;
var text = ubb;
if (textEl.createTextRange && textEl.caretPos) {
   var caretPos = textEl.caretPos;
   caretPos.text = caretPos.text.charAt(caretPos.text.length - 1) == ' ' ? text + ' ' : text;
 }
 else {
   textEl.value = textEl.value + text; }
} 

 function showfacea()
{ 
 var strhtml ="<div id='SmohanFaceBoxa'><span class='Cornera'></span><div class='Content'><h3><span>常用表情</span><a class='close' href='javascript:hiddenfacea(1);' title='关闭'></a></h3><ul><img src='/xml/Template/default/img/loads.gif' alt='.'/>正在加载中...</ul></div></div>" ;
document.getElementById('showfacelista').innerHTML = strhtml; myajax('GET','/xml/bbs/ajax.aspx?aa=2','bbsfacea');
}  
 function hiddenfacea(a) { if(a==1){
document.getElementById('SmohanFaceBoxa').style.display = 'none'; 
}else {
document.getElementById('SmohanFaceBoxa').style.display = 'block';
document.getElementById('SmohanFaceBox').style.display = 'none'; 
document.getElementById('SmohanFaceBox6').style.display = 'none'; 
document.getElementById('SmohanFaceBoxb').style.display = 'none';
document.getElementById('SmohanFaceBoxc').style.display = 'none';
document.getElementById('SmohanFaceBoxd').style.display = 'none';
} }
 function addubba(ubba) {
var textEl = f.content;
var text = ubba;
if (textEl.createTextRange && textEl.caretPos) {
   var caretPos = textEl.caretPos;
   caretPos.text = caretPos.text.charAt(caretPos.text.length - 1) == ' ' ? text + ' ' : text;
 }
 else {
   textEl.value = textEl.value + text; }
} 

 function showfaceb()
{ 
 var strhtml ="<div id='SmohanFaceBoxb'><span class='Cornerb'></span><div class='Content'><h3><span>常用表情</span><a class='close' href='javascript:hiddenfaceb(1);' title='关闭'></a></h3><ul><img src='/xml/Template/default/img/loads.gif' alt='.'/>正在加载中...</ul></div></div>" ;
document.getElementById('showfacelistb').innerHTML = strhtml; myajax('GET','/xml/bbs/ajax.aspx?aa=3','bbsfaceb');
}  
 function hiddenfaceb(a) { if(a==1){ document.getElementById('SmohanFaceBoxb').style.display = 'none'; }else {
document.getElementById('SmohanFaceBoxb').style.display = 'block';
document.getElementById('SmohanFaceBox').style.display = 'none'; 
document.getElementById('SmohanFaceBoxa').style.display = 'none'; 
document.getElementById('SmohanFaceBox6').style.display = 'none';
document.getElementById('SmohanFaceBoxc').style.display = 'none';
document.getElementById('SmohanFaceBoxd').style.display = 'none';} }
 function addubbb(ubbb) {
var textEl = f.content;
var text = ubbb;
if (textEl.createTextRange && textEl.caretPos) {
   var caretPos = textEl.caretPos;
   caretPos.text = caretPos.text.charAt(caretPos.text.length - 1) == ' ' ? text + ' ' : text;
 }
 else {
   textEl.value = textEl.value + text; }
} 

 function showfacec()
{ 
 var strhtml ="<div id='SmohanFaceBoxc'><span class='Cornerc'></span><div class='Content'><h3><span>常用表情</span><a class='close' href='javascript:hiddenfacec(1);' title='关闭'></a></h3><ul><img src='/xml/Template/default/img/loads.gif' alt='.'/>正在加载中...</ul></div></div>" ;
document.getElementById('showfacelistc').innerHTML = strhtml; myajax('GET','/xml/bbs/ajax.aspx?aa=4','bbsfacec');
}  
 function hiddenfacec(a) { if(a==1){ document.getElementById('SmohanFaceBoxc').style.display = 'none'; }else {
document.getElementById('SmohanFaceBoxc').style.display = 'block';
document.getElementById('SmohanFaceBox').style.display = 'none'; 
document.getElementById('SmohanFaceBoxa').style.display = 'none'; 
document.getElementById('SmohanFaceBoxb').style.display = 'none';
document.getElementById('SmohanFaceBox6').style.display = 'none';
document.getElementById('SmohanFaceBoxd').style.display = 'none';
} }
 function addubbc(ubbc) {
var textEl = f.content;
var text = ubbc;
if (textEl.createTextRange && textEl.caretPos) {
   var caretPos = textEl.caretPos;
   caretPos.text = caretPos.text.charAt(caretPos.text.length - 1) == ' ' ? text + ' ' : text;
 }
 else {
   textEl.value = textEl.value + text; }
} 



 function showfaced()
{ 
 var strhtml ="<div id='SmohanFaceBoxd'><span class='Cornerd'></span><div class='Content'><h3><span>常用表情</span><a class='close' href='javascript:hiddenfaced(1);' title='关闭'></a></h3><ul><img src='/xml/Template/default/img/loads.gif' alt='.'/>正在加载中...</ul></div></div>" ;
document.getElementById('showfacelistd').innerHTML = strhtml; myajax('GET','/xml/bbs/ajax.aspx?aa=5','bbsfaced');
}  
 function hiddenfaced(a) { if(a==1){ document.getElementById('SmohanFaceBoxd').style.display = 'none'; }else {
document.getElementById('SmohanFaceBoxd').style.display = 'block';
document.getElementById('SmohanFaceBox').style.display = 'none'; 
document.getElementById('SmohanFaceBoxa').style.display = 'none'; 
document.getElementById('SmohanFaceBoxb').style.display = 'none';
document.getElementById('SmohanFaceBoxc').style.display = 'none';
document.getElementById('SmohanFaceBox6').style.display = 'none';
} }
 function addubbd(ubbd) {
var textEl = f.content;
var text = ubbd;
if (textEl.createTextRange && textEl.caretPos) {
   var caretPos = textEl.caretPos;
   caretPos.text = caretPos.text.charAt(caretPos.text.length - 1) == ' ' ? text + ' ' : text;
 }
 else {
   textEl.value = textEl.value + text; }
} 

function openLogin(){
document.getElementById("win").style.display="";
document.getElementById("wina").style.display="none";
}
function closeLogin(){
document.getElementById("win").style.display="none";
}

$(function(){
  var oBtn = $('#showab');
  var popWindow = $('.popWindow');
  var oClose = $('.popWindow h3 span');
  var browserWidth = $(window).width();
  var browserHeight = $(window).height();
  var browserScrollTop = $(window).scrollTop();
  var browserScrollLeft = $(window).scrollLeft();
  var popWindowWidth = popWindow.outerWidth(true);
  var popWindowHeight = popWindow.outerHeight(true);
  var positionLeft = browserWidth/2 - popWindowWidth/2+browserScrollLeft;
  var positionTop = browserHeight/2 - popWindowHeight/2+browserScrollTop;
  var maskWidth = $(document).width();
  var maskHeight = $(document).height();
  oBtn.click(function(){
    popWindow.show().animate({
          'left':positionLeft+'px',
    },500);
  });
  $(window).resize(function(){
    if(popWindow.is(':visible')){
      browserWidth = $(window).width();
      browserHeight = $(window).height();
      positionLeft = browserWidth/2 - popWindowWidth/2+browserScrollLeft;
      positionTop = browserHeight/2 - popWindowHeight/2+browserScrollTop;
      popWindow.animate({
            'left':positionLeft+'px',
      },500);
    }
  });
});
$(function(){
  var oBtn = $('#showabb');
  var popWindow = $('.popWindowa');
  var oClose = $('.popWindowa h3 span');
  var browserWidth = $(window).width();
  var browserHeight = $(window).height();
  var browserScrollTop = $(window).scrollTop();
  var browserScrollLeft = $(window).scrollLeft();
  var popWindowWidth = popWindow.outerWidth(true);
  var popWindowHeight = popWindow.outerHeight(true);
  var positionLeft = browserWidth/2 - popWindowWidth/2+browserScrollLeft;
  var positionTop = browserHeight/2 - popWindowHeight/2+browserScrollTop;
  var maskWidth = $(document).width();
  var maskHeight = $(document).height();
  oBtn.click(function(){
    popWindow.show().animate({
          'left':positionLeft+'px',
    },500);
  });
  $(window).resize(function(){
    if(popWindow.is(':visible')){
      browserWidth = $(window).width();
      browserHeight = $(window).height();
      positionLeft = browserWidth/2 - popWindowWidth/2+browserScrollLeft;
      positionTop = browserHeight/2 - popWindowHeight/2+browserScrollTop;
      popWindow.animate({
            'left':positionLeft+'px',
      },500);
    }
  });
});

function openLogina(){
document.getElementById("wina").style.display="";
document.getElementById("win").style.display="none";
}
function closeLogina(){
document.getElementById("wina").style.display="none";
}
 
function fnTrue(num,messageid){
if(messageid=='bbsfacea') {
if(num.indexOf("SmohanFaceBoxa") > 0){ document.getElementById('showfacelista').innerHTML = num; }
}
if(messageid=='bbsfaceb') {
if(num.indexOf("SmohanFaceBoxb") > 0){ document.getElementById('showfacelistb').innerHTML = num; }
}
if(messageid=='bbsfacec') {
if(num.indexOf("SmohanFaceBoxc") > 0){ document.getElementById('showfacelistc').innerHTML = num; }
}
if(messageid=='bbsfaced') {
if(num.indexOf("SmohanFaceBoxd") > 0){ document.getElementById('showfacelistd').innerHTML = num; }
}
if(messageid=='bbsface') {
if(num.indexOf("SmohanFaceBox") > 0){ document.getElementById('showfacelist').innerHTML = num; }
}
if(messageid=='bbsface6') {
if(num.indexOf("SmohanFaceBox6") > 0){ document.getElementById('showfhg').innerHTML = num; }
}
if(messageid=='bbsfac1') {
if(num.indexOf("SmohanFaceBox1") > 0){ document.getElementById('showfacelist1').innerHTML = num; }
}
if(messageid=='bbsfac2') {
if(num.indexOf("SmohanFaceBox2") > 0){ document.getElementById('showfacelist2').innerHTML = num; }
}
if(messageid=='bbsfacm') {
if(num.indexOf("SmohanFaceBoxm") > 0){ document.getElementById('showfacelistm').innerHTML = num; }
}
if(messageid=='bbsfack') {
if(num.indexOf("SmohanFaceBoxk") > 0){ document.getElementById('showfacelistk').innerHTML = num; }
}
if(messageid=='bbsf') {
if(num.indexOf("SmohanFaceBoxm") > 0){ document.getElementById('showfac').innerHTML = num; }
}
if(messageid=='bbsfac1a') {
if(num.indexOf("SmohanFaceBox1a") > 0){ document.getElementById('showfacelist1a').innerHTML = num; }
}
if(messageid=='regist') {
if(num.indexOf("regista") > 0){ document.getElementById('comment-list').innerHTML = num; } //注册表单
}
if(messageid=='bbsfackjh') {
if(num.indexOf("SmohanFajgh") > 0){ document.getElementById('showfa123').innerHTML = num; }
}
}
 function fnFalse(messageid){showBox('网络不给力，提交失败！');}


$(function(){
  var oBtn = $('#showabc');
  var popWindow = $('.popWindow');
  var oClose = $('.popWindow h3 span');
  var browserWidth = $(window).width();
  var browserHeight = $(window).height();
  var browserScrollTop = $(window).scrollTop();
  var browserScrollLeft = $(window).scrollLeft();
  var popWindowWidth = popWindow.outerWidth(true);
  var popWindowHeight = popWindow.outerHeight(true);
  var positionLeft = browserWidth/2 - popWindowWidth/2+browserScrollLeft;
  var positionTop = browserHeight/2 - popWindowHeight/2+browserScrollTop;
  var maskWidth = $(document).width();
  var maskHeight = $(document).height();
  oBtn.click(function(){
    popWindow.show().animate({
          'left':positionLeft+'px',
    },500);
  });
  $(window).resize(function(){
    if(popWindow.is(':visible')){
      browserWidth = $(window).width();
      browserHeight = $(window).height();
      positionLeft = browserWidth/2 - popWindowWidth/2+browserScrollLeft;
      positionTop = browserHeight/2 - popWindowHeight/2+browserScrollTop;
      popWindow.animate({
            'left':positionLeft+'px',
      },500);
    }
  });
});


﻿




﻿$(document).ready(function () {
$("#xshadx").bind("click",function(ev){
          var id = $('#id').val().trim();
                var uid = $('#uid').val().trim();
                var xla = $('#xla').val().trim();
         var money = $('#money').val().trim();
 if (isNaN(xla) || xla < 1 || xla > 99999999) {
                 showBox('输入数量大于1小于99999999')
}
else {
myVar = setTimeout(myFunction, 5000)
function myFunction() {
  showBox('网络堵塞,正在拼命加载!')
}
mar = setTimeout(myF, 10000)
function myF() {
  showBox('堵塞超时!请刷新页面')
document.getElementById("win").style.display="none";
$('#xshadx').removeAttr('disabled').val('奖励主播').css('background-color', '');
}
$('#xshadx').attr('disabled', 'true').val('奖赏中...').css('background-color', '');
$.getScript('/bbs/chat/yy.aspx?id=' + id + '&xl=' + xla + '&money=' + money + '&uid=' + uid + '', function () {
var status = user.status;
      clearTimeout(myVar)
      clearTimeout(mar)
document.getElementById("win").style.display="none";
 if (status == 0) {
                  showBox('奖赏成功!')
$('#xshadx').removeAttr('disabled').val('奖励主播').css('background-color', '');
 }
 else {
$('#xshadx').removeAttr('disabled').val('奖励主播').css('background-color', '');
 if (status == 1) {
      showBox('请登录在打赏!')
    }
   else {
   if (status == 2) {
      showBox('您的货币不足!')
    }
   else {
    if (status == 3) {
      showBox('您是主播,不能打赏自己!')
    }
   else {
   if (status == 4) {
      showBox('不是主播.不能打赏!')
    }
   else {
      showBox('主播出差.请刷新页面!')
  }}}}}
   });
  }
 });
});
