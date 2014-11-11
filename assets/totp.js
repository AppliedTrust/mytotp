
var expires = 30;

$(function() {
  doCode();
});


function doCode() {
  var tail = $.getJSON( "/codes/", function(d) {
    expires = Date.now() + d.Valid*1000;
    setTimeout(function(){doCode()}, d.Valid*1000);
    countdownTimer = (d.Valid-10)*1000;
    if (countdownTimer < 0) {
      countdownTimer = 0;
    }
    setTimeout(function(){doCountdown()}, countdownTimer);
    $("#codes").fadeOut(500, function() {
      d.Codes.forEach(function(code) {
        if ( $("#codes #code-"+code.Id).length ) {
          $("#codes #code-"+code.Id).html(code.Code);
        } else {
          $("#codes").append("<div class='row'>");
          $("#codes").append("<div class='name col-md-8 col-md-offset-1'>"+code.Name+"</div>");
          $("#codes").append("<div id='code-"+code.Id+"' class='code col-md-2'>"+code.Code+"</div>");
          $("#codes").append("</div>");
        }
      });
    }).fadeIn(500);
  })
  .fail(function() {
    $(".countdown").fadeOut(500);
    $("#codes").html("<h2>Error loading codes :(</h2>");
  });

}

function doCountdown() {
  if ( expires - Date.now() < 10*1000 ) {
    //$(".countdown").fadeIn(500);
    $(".countdown").html(Math.round((expires - Date.now())/1000));
    setTimeout(function(){doCountdown()}, 1000);
  } else {
    //$(".countdown").fadeTo(500, 0.01);
    $(".countdown").html("");
  }
}


var start = new Date();
var step=0;
var gradientTime = 30 * 1000
var gradientInterval = 20;
var gradientSpeed = gradientInterval / gradientTime ;

  step += gradientSpeed;
  if ( step >= 1 )
  {
    step = 0 ;
  }

var colors = new Array(
      [62,35,255],
      [60,255,60],
      [255,35,98],
      [45,175,230]
);

$(function() {
//  setInterval(updateGradient,gradientInterval);
});

function updateGradient() {
  console.log("gradientSpeed: "+gradientSpeed);
  console.log("step: "+step);
  console.log("updateGradient() ");
  var c0_0 = colors[0];
  var c0_1 = colors[1];
  var c1_0 = colors[2];
  var c1_1 = colors[3];

  var istep = 1 - step;
  var r1 = Math.round(istep * c0_0[0] + step * c0_1[0]);
  var g1 = Math.round(istep * c0_0[1] + step * c0_1[1]);
  var b1 = Math.round(istep * c0_0[2] + step * c0_1[2]);
  var color1 = "rgb("+r1+","+g1+","+b1+")";

  var r2 = Math.round(istep * c1_0[0] + step * c1_1[0]);
  var g2 = Math.round(istep * c1_0[1] + step * c1_1[1]);
  var b2 = Math.round(istep * c1_0[2] + step * c1_1[2]);
  var color2 = "rgb("+r2+","+g2+","+b2+")";
  $('body').css({
    background: "-webkit-gradient(linear, left top, right top, from("+color1+"), to("+color2+"))"}).css({
    background: "-moz-linear-gradient(left, "+color1+" 0%, "+color2+" 100%)"});
  
}
