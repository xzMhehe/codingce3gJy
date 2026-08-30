

/////////////
//headerfix.js
/////////////




(window.constructQZFL = function(){




//})();



/////////////
//qzfl.js
/////////////


/**
 * @fileOverview QZFL 涓绘鏋堕€昏緫锛?br/>
					Qzone Front-end Library: Liberation<br />
					QZFL 鏄敱绌洪棿骞冲彴寮€鍙戠粍锛屽紑鍙戠殑涓€濂梛s妗嗘灦搴撱€?br />
					QZFL 鏈€鍚庣殑 L 鏈変袱涓剰鎬濓紝鍏朵腑涓€涓剰鎬濇槸 Library 鍔熻兘搴擄紝璇存槑杩欐槸涓€涓墠鍙扮殑妗嗘灦搴?<br />
					鍚屾椂 L 涔熸槸 Liberation 瑙ｆ斁鐨勬剰鎬濓紝杩欐槸甯屾湜閫氳繃 QZFL 鑳芥妸澶у鍦↗S寮€鍙戝伐浣滀腑瑙ｆ斁鍑烘潵銆?					QZFL鍚勭鍚堝苟鐗堟湰閮藉繀椤诲寘鍚湰婧愭枃浠? * @version 2.0.9.6 ($Rev: 1921 $)
 * @author QzoneWebGroup - ($LastChangedBy: ryanzhao $) - ($Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $)
 */

/**
 * QZFL鍚嶅瓧绌洪棿
 * @namespace QZFL鍚嶅瓧绌洪棿
 * @name QZFL
 */
window.QZFL = window.QZONE = window.QZFL || window.QZONE || {};

/**
 * 鐗堟湰鍙疯鏄庡瓧
 * @type string
 */
QZFL.version = "2.1.1.7";

/**
 * 鐗堟湰鍙锋暟瀛? * @public
 * @type number
 */
QZFL._qzfl = 2.117;

/**
 * 瀹氫箟涓€涓€氱敤绌哄嚱鏁? * @returns {undefined}
 */
QZFL.emptyFn = function() {};

/**
 * 瀹氫箟涓€涓€氱敤閫忎紶鍑芥暟
 * @param {number|string|object|function|undefined} [v = undefined]
 * @returns {number|string|object|function|undefined} 灏辨槸浼犲叆鐨剉鐩存帴閫忎紶鍑烘潵
 */
QZFL.returnFn = function(v) {
	return v;
};

/**
 * 娴忚鍣ㄥ垽鏂紩鎿庯紝缁欑▼搴忔彁渚涙祻瑙堝櫒鍒ゆ柇鐨勬帴鍙? * @namespace 娴忚鍣ㄥ垽鏂紩鎿? * @name userAgent
 * @memberOf QZFL
 */
(function(){
	var ua = QZFL.userAgent = {}
		, agent = navigator.userAgent
		, nv = navigator.appVersion
		, r
		, m
		, optmz;

	/**
	 * 璋冩暣娴忚鍣ㄧ殑榛樿琛屼负锛屼娇涔嬩紭鍖?	 * @deprecated 宸茬粡涓嶅缓璁樉寮忚皟鐢ㄤ簡锛岀敱QZFL鍒濆鍖栨椂璋冪敤
	 * @function
	 * @static
	 * @name adjustBehaviors
	 * @memberOf QZFL.userAgent
	 */
	ua.adjustBehaviors = QZFL.emptyFn;
	
	if (window.ActiveXObject || window.msIsStaticHTML) {//ie (document.querySelectorAll) or IE11
		/**
		 * IE鐗堟湰鍙凤紝濡傛灉涓嶆槸IE锛屾鍊间负 NaN
		 * @field
		 * @type number
		 * @static
		 * @name ie
		 * @memberOf QZFL.userAgent
		 */
		ua.ie = 6;

		(window.XMLHttpRequest || (agent.indexOf('MSIE 7.0') > -1)) && (ua.ie = 7); 
		(window.XDomainRequest || (agent.indexOf('Trident/4.0') > -1)) && (ua.ie = 8); 
		(agent.indexOf('Trident/5.0') > -1) && (ua.ie = 9); 
		(agent.indexOf('Trident/6.0') > -1) && (ua.ie = 10); 
		(agent.indexOf('Trident/7.0') > -1) && (ua.ie = 11);	

		/**
		 * 褰撳墠鐨処E娴忚鍣ㄦ槸鍚︿负beta鐗堟湰
		 * @field
		 * @type boolean
		 * @static
		 * @name isBeta
		 * @memberOf QZFL.userAgent
		 */
		ua.isBeta = navigator.appMinorVersion && navigator.appMinorVersion.toLowerCase().indexOf('beta') > -1;

		//涓€浜涙祻瑙堝櫒琛屼负鐭
		if (ua.ie < 7) {//IE6 鑳屾櫙鍥惧己鍒禼ache
			try {
				document.execCommand('BackgroundImageCache', false, true);
			} catch (ign) {}
		}

		//鍒涘缓涓€涓猟ocument寮曠敤
		QZFL._doc = document;

		//鎵╁睍IE涓嬩袱涓猻etTime鐨勪紶鍙傝兘鍔?		optmz = function(st){
				return function(fns, tm){
						var aargs;
						if(typeof fns == 'string'){
							return st(fns, tm);
						}else{
							aargs = Array.prototype.slice.call(arguments, 2);
							return st(function(){
									fns.apply(null, aargs);
								}, tm);
						}
					};
			};
		window.setTimeout = QZFL._setTimeout = optmz(window.setTimeout);
		window.setInterval = QZFL._setInterval = optmz(window.setInterval);

	} else if (document.getBoxObjectFor || typeof(window.mozInnerScreenX) != 'undefined') {
		r = /(?:Firefox|GranParadiso|Iceweasel|Minefield).(\d+\.\d+)/i;

		/**
		 * FireFox娴忚鍣ㄧ増鏈彿锛岄潪FireFox鍒欎负 NaN
		 * @field
		 * @type number
		 * @static
		 * @name firefox
		 * @memberOf QZFL.userAgent
		 */
		ua.firefox = parseFloat((r.exec(agent) || r.exec('Firefox/3.3'))[1], 10);
	} else if (!navigator.taintEnabled) {//webkit
		m = /AppleWebKit.(\d+\.\d+)/i.exec(agent);

		/**
		 * Webkit鍐呮牳鐗堟湰鍙凤紝闈濿ebkit鍒欎负 NaN
		 * @field
		 * @type number
		 * @static
		 * @name webkit
		 * @memberOf QZFL.userAgent
		 */
		ua.webkit = m ? parseFloat(m[1], 10) : (document.evaluate ? (document.querySelector ? 525 : 420) : 419);
		
		if ((m = /Chrome.(\d+\.\d+)/i.exec(agent)) || window.chrome) {

			/**
			 * Chrome娴忚鍣ㄧ増鏈彿锛岄潪Chrome娴忚鍣ㄥ垯涓?NaN
			 * @field
			 * @type number
			 * @static
			 * @name chrome
			 * @memberOf QZFL.userAgent
			 */
			ua.chrome = m ? parseFloat(m[1], 10) : '2.0';
		} else if ((m = /Version.(\d+\.\d+)/i.exec(agent)) || window.safariHandler) {

			/**
			 * Safari娴忚鍣ㄧ増鏈彿锛岄潪Safari娴忚鍣ㄥ垯涓?NaN
			 * @field
			 * @type number
			 * @static
			 * @name safari
			 * @memberOf QZFL.userAgent
			 */
			ua.safari = m ? parseFloat(m[1], 10) : '3.3';
		}

		/**
		 * 褰撳墠椤甸潰鏄惁涓篴ir client
		 * @field
		 * @type boolean
		 * @static
		 * @name air
		 * @memberOf QZFL.userAgent
		 */
		ua.air = agent.indexOf('AdobeAIR') > -1 ? 1 : 0;

		/**
		 * 鏄惁涓篿Pod瀹㈡埛绔〉闈?		 * @field
		 * @type boolean
		 * @static
		 * @name isiPod
		 * @memberOf QZFL.userAgent
		 */
		ua.isiPod = agent.indexOf('iPod') > -1;

		/**
		 * 鏄惁涓篿Pad瀹㈡埛绔〉闈?		 * @field
		 * @type boolean
		 * @static
		 * @name isiPad
		 * @memberOf QZFL.userAgent
		 */
		ua.isiPad = agent.indexOf('iPad') > -1;

		/**
		 * 鏄惁涓篿Phone瀹㈡埛绔〉闈?		 * @field
		 * @type boolean
		 * @static
		 * @name isiPhone
		 * @memberOf QZFL.userAgent
		 */
		ua.isiPhone = agent.indexOf('iPhone') > -1;
	} else if (window.opera) {//opera

		/**
		 * Opera娴忚鍣ㄧ増鏈彿锛岄潪Opera鍒欎负 NaN
		 * @field
		 * @type number
		 * @static
		 * @name opera
		 * @memberOf QZFL.userAgent
		 */
		ua.opera = parseFloat(window.opera.version(), 10);
	} else {//榛樿IE6鍚?		ua.ie = 6;
	}	

	/**
	 * 鏄惁涓篗acOS
	 * @field
	 * @type boolean
	 * @static
	 * @name macs
	 * @memberOf QZFL.userAgent
	 */
	if (!(ua.macs = agent.indexOf('Mac OS X') > -1)) {

		/**
		 * Windows鎿嶄綔绯荤粺鐗堟湰鍙凤紝涓嶆槸鐨勮瘽涓篘aN
		 * @field
		 * @type number
		 * @static
		 * @name windows
		 * @memberOf QZFL.userAgent
		 */
		ua.windows = ((m = /Windows.+?(\d+\.\d+)/i.exec(agent)), m && parseFloat(m[1], 10));

		/**
		 * 鏄惁Linux鎿嶄綔绯荤粺锛屼笉鏄殑璇濅负false
		 * @field
		 * @type boolean
		 * @static
		 * @name linux
		 * @memberOf QZFL.userAgent
		 */
		ua.linux = agent.indexOf('Linux') > -1;

		/**
		 * 鏄惁Android鎿嶄綔绯荤粺锛屼笉鏄殑璇濅负false
		 * @field
		 * @type boolean
		 * @static
		 * @name android
		 * @memberOf QZFL.userAgent
		 */
		ua.android = agent.indexOf('Android') > -1;
	}


	/**
	 * 鏄惁iOS鎿嶄綔绯荤粺锛屼笉鏄殑璇濅负false
	 * @field
	 * @type boolean
	 * @static
	 * @name iOS
	 * @memberOf QZFL.userAgent
	 */
	ua.iOS = agent.indexOf('iPhone OS') > -1;
	!ua.iOS && (m = /OS (\d+(?:_\d+)*) like Mac OS X/i.exec(agent), ua.iOS = m && m[1] ? true : false);


})();

/**
 * QZFL瀵笿avascript Object鐨勬帴鍙ｅ皝瑁咃紝鎻愪緵涓€浜涘師鐢熻兘鍔? * @namespace 瀵硅薄鍩虹澶勭悊
 */
QZFL.object = {

	/**
	 * 鎶婂懡鍚嶇┖闂寸殑鏂规硶鏄犲皠鍒板叏灞€
	 * @param {object} object 瀵硅薄
	 * @param {object} [scope=window] 鐩爣绌洪棿
	 * @deprecated 涓嶆帹鑽愬父浣跨敤锛岄伩鍏嶅彉閲忓悕鍐茬獊

	 * @example
	 * QZFL.object.map(QZFL.lang)
	 */
	map : function(object, scope) {
		return QZFL.object.extend(scope || window, object);
	},

	/**
	 * 鍛藉悕绌洪棿鍔熻兘鎵╁睍
	 * @param {object} namespace 闇€瑕佽鎵╁睍鐨勫懡鍚嶇┖闂?	 * @param {object} extendModule 闇€瑕佹墿灞曠殑鍔熻兘鍖?	 * @returns {object} 杩斿洖琚墿灞曠殑鍛藉悕绌洪棿锛屽嵆鎵╁睍鍚庣殑namespace

	 * @example
	 * QZFL.object.extend(QZFL.dialog, { fn1: function(){} } );
	 */
	extend : function() {
		var args = arguments,
			len = arguments.length,
			deep = false,
			i = 1,
			target = args[0],
			opts,
			src,
			clone,
			copy;

		if ( typeof target === "boolean" ) {
			deep = target;
			target = arguments[1] || {};
			i = 2;
		}

		if ( typeof target !== "object" && typeof target !== "function" ) {
			target = {};
		}

		if ( len === i ) {
			target = QZFL;
			--i;
		}

		for ( ; i < len; i++ ) {
			if ( (opts = arguments[ i ]) != null ) {
				for (var name in opts ) {
					src = target[ name ];
					copy = opts[ name ];

					if ( target === copy ) {
						continue;
					}

					if ( deep && copy && typeof copy === "object" && !copy.nodeType ) {

						if ( src ) {
							clone = src;
						} else if ( QZFL.lang.isArray(copy) ) {
							clone = [];
						} else if ( QZFL.object.getType(copy) === 'object' ) {
							clone = {};
						} else {
							clone = copy;
						}

						target[ name ] = QZFL.object.extend( deep, clone, copy );

					} else if ( copy !== undefined ) {
						target[ name ] = copy;
					}
				}
			}
		}

		return target;
	},
	
	/**
	 * 瀵瑰璞℃垚鍛樻壒閲忔墽琛屼竴涓搷浣?	 *
	 * @param {object} obj 琚搷浣滃璞″璞?	 * @param {function} callback 鎵€鎵ц鐨勬搷浣?	 * @returns {object} 浼犲叆鐨刼bj瀵硅薄

	 * @example
	 * QZFL.object.each([1,2,3], function(){ alert(this) } );
	 */
	each : function(obj, callback) {
		var value,
			i = 0,
			length = obj.length,
			isObj = (length === undefined) || (typeof(obj)=="function");
		if (isObj) {
			for (var name in obj) {
				if (callback.call(obj[name], obj[name], name, obj) === false) {
					break;
				}
			}
		} else {
			for (value = obj[0]; i < length && false !== callback.call(value, value, i, obj); value = obj[++i]) { }
		}
		return obj;
	},

	/**
	 * 鑾峰彇瀵硅薄绫诲瀷
	 *
	 * @param {object} obj 浠绘剰涓€涓暟鎹?	 * @return {string} 杩斿洖瀵硅薄绫诲瀷瀛楃涓?
	 * @example
	 * QZFL.object.getType([1,2,3]);
	 */
	getType : function(obj) {
		return obj === null ? 'null' : (obj === undefined ? 'undefined' : Object.prototype.toString.call(obj).slice(8, -1).toLowerCase());
	},
	
	/**
	 * route鐢ㄥ埌鐨勬鍒欏璞?	 * @field
	 * @static
	 * @private
	 * @type RegExp
	 * @default /([\d\w_]+)/g
	 */
	routeRE : /([\d\w_]+)/g,
	
	/**
	 * 鐢ㄥ璞¤矾寰勫彇涓€涓狫SON瀵硅薄涓殑瀛愬璞″紩鐢?	 * @static
	 * @param {object} obj 婧愬璞?	 * @param {string} path 瀵硅薄鑾峰彇璺緞
	 * @returns {object|string|number|function}

	 * @example
	 * QZFL.object.route(
	           { "a" : 
			       { "b" :
				       { "c" : "Hello World"
					   }
			       }
			   },
			   "a.b.c"
	       ); //杩斿洖鍊硷細"Hello World"
	 */
	route: function(obj, path){
		obj = obj || {};
		path = String(path);

		var r = QZFL.object.routeRE,
			m;
	
		r.lastIndex = 0;

		while ((m = r.exec(path)) !== null) {
			obj = obj[m[0]];
			if (obj === undefined || obj === null) {
				break;
			}
		}

		return obj;
	},
	
	/**
	 * 灏嗘柟娉曠粦瀹氬湪瀵硅薄涓婏紝鑳藉淇濇姢this鎸囬拡涓嶄細鈥滄紓绉烩€?	 * @param {object} obj 姣嶄綋瀵硅薄
	 * @param {object} fn 鐩爣鏂规硶

	 * @example var e = QZFL.event.bind(objA, funB);
	 */
	bind : function(obj, fn) {
		var slice = Array.prototype.slice,
			args = slice.call(arguments, 2);

		return function(){
			obj = obj || this;
			fn = typeof fn == 'string' ? obj[fn] : fn;
			fn = typeof fn == 'function' ? fn : QZFL.emptyFn;
			return fn.apply(obj, args.concat(slice.call(arguments, 0)));
		};
	},

	/**
	 * 鎶婃寚瀹氬懡鍚嶇┖闂翠笅鐨勬柟娉?浠ョ煭鍚嶇殑鏂瑰紡 鏄犲皠鍒板彟涓€涓懡鍚嶇┖闂?	 * @param {object} src 婧愬璞?	 * @param {object} tar 鐩爣瀵硅薄
	 * @param {function} [rule=function(name){ return '$' + name; }] 鏄犲皠鍚嶅瓧鐨勫鐞嗗櫒
	 * @returns {undefined}
	 */
	ease : function(src, tar, rule){
		if (tar) {
			if (typeof(rule) != 'function') {
				rule = QZFL.object._eachFn;
			}
			QZFL.object.each(src, function(v, k){
				if (typeof(v) == 'function') {
					tar[rule(k)] = v;
				}
			});
		}
	},
	
	/**
	 * QZFL.object.ease 鐨勫懡鍚嶆槧灏勯粯璁ゆ柟妗?	 * @param {string} name 婧愬悕
	 * @returns {string} 杞崲鍚庣殑鍚嶅瓧
	 * @private
	 */
	_easeFn : function(name){
		return '$' + name;
	}
};



/**
 * QZFL瀵笿avascript Object鐨勬帴鍙ｅ皝瑁咃紝鎻愪緵涓€浜涘師鐢熻兘鍔?<strong style="color:red;">Deprecated</strong>
 * @namespace 瀵硅薄鍩虹澶勭悊
 * @deprecated 寤鸿涓嶈鐢ㄨ繖涓簡锛岀敤QZFL.object涓殑鐩稿叧鏂规硶
 * @see QZFL.object
 */
QZFL.namespace = QZFL.object;

/**
 * QZFL 璋冭瘯寮曟搸鎺ュ彛锛屼负璋冭瘯鎻愪緵鎺ュ叆鐨勫彲鑳? * @namespace QZFL璋冭瘯寮曟搸鎺ュ彛
 */
QZFL.runTime = {

	/**
	 * 鏄惁澶勪簬debug妯″紡
	 * @field
	 * @static
	 * @type boolean
	 * @default false
	 */
	isDebugMode : false,

	/**
	 * 閿欒鎶ュ憡鎺ュ彛
	 * @function
	 * @static
	 * @type function
	 * @default QZFL.emptyFn
	 */
	error : QZFL.emptyFn,

	/**
	 * 璀﹀憡淇℃伅鎶ュ憡鎺ュ彛
	 * @function
	 * @static
	 * @type function
	 * @default QZFL.emptyFn
	 */
	warn : QZFL.emptyFn
};

//---------------------------------------------

/**
 * qzfl 鎺у埗鍙帮紝鐢ㄤ簬鏄剧ず璋冭瘯淇℃伅浠ュ強杩涜涓€浜涚畝鍗曠殑鑴氭湰璋冭瘯绛夋搷浣? * @namespace QZFL鎺у埗鍙版帴鍙ｏ紝鐢ㄦ潵鏄剧ず绋嬪簭杈撳嚭鐨刲og淇℃伅銆? */
QZFL.console = window.console || {};//function(expr){};

/**
 * 鍦╟onsole閲屾樉绀轰俊鎭? * @returns {undefined}
 */
QZFL.console.log = QZFL.console.log || function() {};

/**
 * 鍦╟onsole閲屾樉绀轰俊鎭? * @deprecated
 * @returns {undefined}
 */
QZFL.console.print = QZFL.console.log;

//----------------------------------------------

/**
 * 鍚勭鍔熻兘鍚勫紓鐨勭粍浠朵唬鐮? * @namespace QZFL灏忕粍浠跺寘
 *
 */
QZFL.widget = {};

//----------------------------------------------

//鎶奞ZFL.object涓嬬殑鏂瑰紡鐩存帴鏄犲皠鍒癚ZFL鍛藉悕绌洪棿涓?QZFL.object.map(QZFL.object, QZFL);



/////////////
//qzone_config.js
/////////////





/**
 * @author scorr
 */



(function(w){
	QZFL.config = QZFL.config || {};

	var preFix, cw = w;
	do{
		try{
			var a = null,
				domain;
			if(cw.siDomain){
				domain = cw.siDomain;
				if(domain.indexOf('http://') === -1 &&
					domain.indexOf('https://') === -1){
                    domain = '//' + domain;
				}
                a = document.createElement('a');
                a.href = domain;
                QZFL.config.resourceDomain = a.hostname;
			}
            if(cw.imgcacheDomain){
                domain = cw.imgcacheDomain;
                if(domain.indexOf('http://') === -1 &&
                    domain.indexOf('https://') === -1){
                    domain = '//' + domain;
                }
                a = document.createElement('a');
                a.href = domain;
                QZFL.config.domain = a.hostname;
            }
		}catch(err){ //椤甸潰璺ㄥ煙灏辩畻浜?			break;
		}
	}while((cw !== cw.parent) && (cw = cw.parent));
	
	QZFL.config.defaultMediaRate = 2;
})(window);



/////////////
//config.js
/////////////


/**
 * @fileoverview QZFL鍏ㄥ眬閰嶇疆鏂囦欢
 * @version $Rev: 1921 $
 * @author QzoneWebGroup - ($LastChangedBy: ryanzhao $) - ($Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $)
 */



/**
 * QZFL閰嶇疆锛岀敤鏉ュ瓨鍌≦ZFL涓€浜涚粍浠堕渶瑕佺殑鍙傛暟
 * @namespace QZFL閰嶇疆
 */
QZFL.config = QZFL.config || {};

/**
 * 璋冭瘯绛夌骇
 * @type number
 * @default 0
 */
(typeof QZFL.config.debugLevel == 'undefined') && (QZFL.config.debugLevel = 0);

/**
 * 榛樿涓庡悗鍙颁氦浜掔殑缂栫爜
 *
 * @type string
 * @default "GB2312"
 */
(typeof QZFL.config.defaultDataCharacterSet == 'undefined') && (QZFL.config.defaultDataCharacterSet = "GB2312");

/**
 * dataCenter涓璫ookie瀛樺偍鐨勯粯璁ゅ煙鍚? *
 * @type string
 * @default "qzone.qq.com"
 */
(typeof QZFL.config.DCCookieDomain == 'undefined') && (QZFL.config.DCCookieDomain = "qzone.qq.com");

/**
 * 绯荤粺榛樿涓€绾у煙鍚? *
 * @type string
 * @default "qq.com"
 */
(typeof QZFL.config.domainPrefix == 'undefined') && (QZFL.config.domainPrefix = "qq.com");

/**
 * 榛樿涓诲煙鍚? *
 */
(typeof QZFL.config.domain == 'undefined') && (QZFL.config.domain = "qzs.qq.com");

if(location.hostname.indexOf("qzone.qq.com")>=0){
	(typeof QZFL.config.domainPrefix == 'undefined') && (QZFL.config.domainPrefix = "qzone.qq.com");
	(typeof QZFL.config.domain == 'undefined') && (QZFL.config.domain = "qzs.qzone.qq.com");
}

/**
 * 榛樿cookie free涓诲煙鍚? *
 */
(typeof QZFL.config.resourceDomain == 'undefined') && (QZFL.config.resourceDomain = "qzonestyle.gtimg.cn");




/**
 * XHR proxy鐨刧bencoder dictionary璺緞(闇€瑕佸鍐?
 *
 * @type string
 * @default "http://qzs.qq.com/qzone/v5/toolpages/"
 */
QZFL.config.gbEncoderPath = location.protocol + "//" + QZFL.config.domain + "/qzone/v5/toolpages/";

/**
 * FormSender鐨刪elper page(闇€瑕佸鍐?
 *
 * @type string
 * @default "http://qzs.qq.com/qzone/v5/toolpages/fp_gbk.html"
 */
QZFL.config.FSHelperPage = "//" + QZFL.config.domain + "/qzone/v5/toolpages/fp_gbk.html";

/**
 * 榛樿flash ShareObject鍦板潃
 * @type string
 * @default "http://qzs.qq.com/qzone/v5/toolpages/getset.swf"
 */
QZFL.config.defaultShareObject = location.protocol + "//" + QZFL.config.resourceDomain + "/qzone/v5/toolpages/getset.swf";

/**
 * 榛樿闈欐€侀〉鐨剆erver鍦板潃
 * @type string
 * @default "http://qzs.qq.com/ac/qzone/qzfl/lc/"
 */
QZFL.config.staticServer = location.protocol + "//" + QZFL.config.resourceDomain + "/ac/qzone/qzfl/lc/";




/////////////
//css.js
/////////////


/**
 * @fileoverview QZFL鏍峰紡澶勭悊,鎻愪緵澶氭祻瑙堝櫒鍏煎鐨勬牱寮忚〃澶勭悊
 * @version $Rev: 1921 $
 * @author QzoneWebGroup ($LastChangedBy: ryanzhao $) - $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */



/**
 * QZFL css 宸ュ叿鍖咃紝缁欐祻瑙堝櫒鎻愪緵鍩烘湰鐨勬牱寮忓鐞嗘帴鍙? *
 * @namespace QZFL css 宸ュ叿鍖? */
QZFL.css = {
	/**
	 * 鐢ㄤ互鍖归厤鏍峰紡绫诲悕鐨勬鍒欐睜
	 * @private
	 * @ignore
	 * @type object
	 */
	classFileNameCache: {},

	/**
	 * 鑾峰彇鐢ㄤ互鍖归厤鏍峰紡绫诲悕鐨勬鍒?	 *
	 * @param {string} className 鏍峰紡鍚嶇О
	 * @returns {RegExp} 鐢ㄤ互鍖归厤鐨勬鍒欒〃杈惧紡瑙勫垯
	 * @private
	 * @deprecated 娌″暐澶х敤
	 *
	getClassRegEx: function(className){
		var o = QZFL.css.classNameCache;
		return o[className] || (o[className] = new RegExp('(?:^|\\s+)' + className + '(?:\\s+|$)'));
	},*/

	/**
	 * 鎶?6杩涘埗鐨勯鑹茶浆鎹㈡垚10杩涘埗棰滆壊鐨勫伐鍏?	 * @param {string} color 鍗佸叚杩涘埗棰滆壊鏂囨湰
	 * @returns {number[]} 杩斿洖鏁扮粍褰㈠紡鐨?0杩涘埗棰滆壊

	 * @example QZFL.css.convertHexColor("#ff00ff") //[255, 0, 255];
	 */
	convertHexColor: function(color){
		color = String(color || '');
		color.charAt(0) == '#' && (color = color.substring(1));
		color.length == 3 && (color = color.replace(/([0-9a-f])/ig, '$1$1'));
		return color.length == 6 ? [parseInt(color.substr(0, 2), 16), parseInt(color.substr(2, 2), 16), parseInt(color.substr(4, 2), 16)] : [0, 0, 0];
	},

	/**
	 * 浠嶳GB鑹插僵妯″瀷鍒癏SL鑹插僵妯″瀷鐨勮浆鎹㈢畻娉?	 * @param {number} r 绾㈣壊鍊硷紝10杩涘埗鏁存暟锛孾0, 255]
	 * @param {number} g 缁胯壊鍊硷紝10杩涘埗鏁存暟锛孾0, 255]
	 * @param {number} b 钃濊壊鍊硷紝10杩涘埗鏁存暟锛孾0, 255]
	 * @return {object} result.h 鏄壊鐩稿€?[0, 360) 搴︼紱result.s 鏄ケ鍜屽害鍊糩0, 1]锛況esult.l 鏄寒搴﹀€糩0, 1]
	 */
	rgb2hsl: function(r, g, b){
		var t
			, red = Math.max(r / 255, 0)
			, green = Math.max(g / 255, 0)
			, blue = Math.max(b / 255, 0)
			, max = Math.max(red, green, blue)
			, min = Math.min(red, green, blue)
			, result = {
					h : 0
					, s : 0
					, l : Math.max((max + min) / 2, 0)
				};
		
		if(max != min){
			if(max == red){
				result.h = (t = 60 * ((green - blue) / (max - min))) < 0 ? (t + 360) : t;
			}else if(max == green){
				result.h = (60 * ((blue - red) / (max - min)) + 120);
			}else if(max == blue){
				result.h = (60 * ((red - green) / (max - min)) + 240);
			}

			if(result.l <= 0.5){
				result.s = (max - min) / (2 * result.l);
			}else if(0.5 < result.l){
				result.s = (max - min) / (2 - 2 * result.l);
			}

			result.h = Math.round(result.h);
			result.s = Math.round(result.s * 100) / 100;
			result.l = Math.round(result.l * 100) / 100;
		}

		return result;
	},


	/**
	 * 缂撳瓨褰撳墠椤甸潰鐨勬牱寮忚〃瀵硅薄寮曠敤鐨勬睜
	 *
	 * @private
	 * @type object
	 * @deprecated 涓嶈鍐嶇敤浜?	 *
	styleSheets: {},*/
	
	/**
	 * 閫氳繃id鍙疯幏鍙栨牱寮忚〃
	 * @param {string|number} id 鏍峰紡琛ㄧ殑缂栧彿
	 * @returns {object} 杩斿洖鏍峰紡琛ㄥ璞★紝娌℃湁鍖归厤鍒欎负null

	 * @example QZFL.css.getStyleSheetById("div_id");
	 */
	getStyleSheetById: function(id){
		var s;
		return (s = QZFL.dom.get(id)) && s.sheet || (s = document.styleSheets) && s[id];
	},
	
	/**
	 * 鑾峰彇stylesheet鐨勬牱寮忚鍒?	 * @param {string|number} id 鏍峰紡琛ㄧ殑缂栧彿
	 * @example QZFL.css.getRulesBySheet("css_id");
	 * @returns {object} 杩斿洖鏍峰紡琛ㄨ鍒欓泦鍚堝璞★紝鑻ユ湭鍙戠敓鍖归厤鍒欎负绌哄璞?	 */
	getRulesBySheet: function(sheetId){
		var ss = typeof(sheetId) == "object" ? sheetId : QZFL.css.getStyleSheetById(sheetId),
			rs = {},
			head,
			base;

		if (ss && !(rs = ss.cssRules || ss.rules)) {
			if (head = document.getElementsByTagName('head')[0]) {
				if (base = head.getElementsByTagName('base')[0]) {
					QZFL.dom.removeElement(base);
					rs = ss.cssRules;
					head.appendChild(base);
				}
			}
		}
		return rs;
	},
	
	/**
	 * 鏍规嵁閫夋嫨鍣ㄨ幏寰楁牱寮忚鍒?	 * @param {string} sheetId id 鏍峰紡琛ㄧ殑缂栧彿
	 * @param {string} selector 閫夋嫨鍣ㄥ悕绉?	 * @returns {object} 杩斿洖鍖归厤鍒扮殑鏍峰紡瑙勫垯瀵硅薄锛屾湭鍖归厤鍒欎负null

	 * @example QZFL.css.getRuleBySelector("css_id","#");
	 */
	getRuleBySelector: function(sheetId, selector){
		selector = (String(selector)).toLowerCase();

		var _ss = QZFL.css.getStyleSheetById(sheetId),
			_rs = QZFL.css.getRulesBySheet(_ss);
		
		for (var i = 0, len = _rs.length; i < len; ++i) {
			if (selector == _rs[i].selectorText.toLowerCase()) {
				return _rs[i];
			}
		}
		return null;
	},
	
	/**
	 * 鎻掑叆澶栭摼鏍峰紡琛?	 * @param {string} url 澶栭儴css鍦板潃
	 * @param {string|object} [opts] 鑻ョ被鍨嬩负string鍒欎负link Element鐨処D锛岃嫢涓簅bject鍒欎负鍙€夊弬鏁板寘
	 * @param {string} [opts.linkID=undefined] link Element鐨処D
	 * @param {string} [opts.doc=document] 琚彃鍏ink鑺傜偣鐨勬枃妗ｆ爲鏍?	 * @param {string} [opts.media="screen"] 鏍峰紡鑺傜偣鐨刴edia绉嶇被灞炴€?	 * @returns {object} 杩斿洖鏍峰紡琛ㄥ璞★紝鎻掑叆澶辫触杩斿洖鐨勬槸link Element寮曠敤

	 * @example
QZFL.css.insertCSSLink("/css/style.css", "myCSS1");
QZFL.css.insertCSSLink("/css/style.css", {
	linkID : "myCSS2",
	doc : frames["innerFrame"].document});
	 */
	insertCSSLink: function(url, opts){
		var sid
			, doc
			, t
			, cssLink
			, head
			;

		if(QZFL.css.classFileNameCache[url]){ return; }

		if(typeof opts == "string"){
			sid = opts;
		}

		opts = (typeof opts == "object") ? opts : {};
		sid = opts.linkID || sid;
		doc = opts.doc || document;

		head = doc.getElementsByTagName("head")[0];
		cssLink = ((t = doc.getElementById(sid)) && (t.nodeName == "LINK")) ? t : null;

		if (!cssLink) {
			cssLink = doc.createElement("link");
			sid && (cssLink.id = sid);
			cssLink.rel = cssLink.rev = "stylesheet";
			cssLink.type = "text/css";
			cssLink.media = opts.media || "screen";
			head.appendChild(cssLink);
		}
		try{
			url && (cssLink.href = url);
		}catch(ign){}

		QZFL.css.classFileNameCache[url] = true;

		return (QZFL.userAgent.ie < 9 && cssLink.sheet) || cssLink; //IE>=9寮€濮嬫敮鎸?.sheet浜嗭紝鍜?.styleSheet 鐩稿悓
	},
		
	/**
	 * 鎻掑叆椤甸潰inline鏍峰紡鍧?	 * @param {string} sheetId 鏍峰紡琛╯tyle Element鐨処D
	 * @param {string} [rules=""] 鏍峰紡琛ㄨ鍒欏唴瀹?	 * @returns {object} 杩斿洖鏍峰紡琛╯tyle Element瀵硅薄

	 * @example QZFL.css.insertStyleSheet("cssid", "body {font-size: 75%;}");
	 */
	insertStyleSheet: function(sheetId, rules){
		var node = document.createElement("style");
		node.type = 'text/css';
		sheetId && (node.id = sheetId);
		document.getElementsByTagName("head")[0].appendChild(node);
		if (rules) {
			if (node.styleSheet) {
				node.styleSheet.cssText = rules;
			} else {
				node.appendChild(document.createTextNode(rules));
			}
		}
		return node.sheet || node;
	},
	
	/**
	 * 鍒犻櫎涓€浠芥牱寮忚〃锛屽寘鍚唴閮╯tyle鍜屽閮╟ss
	 * @param {string|number} id 鏍峰紡琛ㄧ殑缂栧彿
	 * @deprecated 瀹炵敤鎬т笉寮猴紝涓嶉€傚悎鍦ㄧ紪绋嬫鏋?
	 * @example QZFL.css.removeStyleSheet("styleid");
	 */
	removeStyleSheet: function(id){
		var _ss = QZFL.css.getStyleSheetById(id);
		_ss && QZFL.dom.removeElement(_ss.owningElement || _ss.ownerNode);
	},
	

	//涓嬮潰寰堟湁鐢ㄧ殑涓€涓鍒欙紝鍏堥潤鎬佸寲鍑烘潵
	_reClassToken: /\s+/,

	/**
	 * 鎿嶄綔鍏冪礌鐨刢lassName鐨勬牳蹇冩柟娉曪紝涔熷彲浠ョ洿鎺ヨ皟鐢紝remove鍙傛暟鏀寔*閫氶厤绗?	 * @param {object} elem 琚搷浣滅殑HTMLElement
	 * @param {string} removeNames 瑕佽鍙栨秷鐨刢lassName浠?	 * @param {string} addNames 瑕佽鍔犲叆鐨刢lassName浠?	 * @returns {string} elem琚搷浣滃悗鐨刢lassName锛岃嫢elem闈炴硶鍒欎负绌轰覆
	 */
	updateClassName: function(elem, removeNames, addNames){
		if (!elem || elem.nodeType != 1) {
			return "";
		}
		var oriName = elem.className,
			_s = QZFL.css,
			ar,
			b; //鍙楀惁鏈夊彉鍖栫殑flag
		if (removeNames && typeof(removeNames) == 'string' || addNames && typeof(addNames) == 'string') {
			if (removeNames == '*') {
				oriName = '';
			} else {
				ar = oriName.split(_s._reClassToken);

				var i = 0,
					l = ar.length,
					n; //涓存椂鍙橀噺

				oriName = {};
				for (; i < l; ++i) { //灏嗗師濮嬬殑className缇ょ粨鏋勫寲涓鸿〃
					ar[i] && (oriName[ar[i]] = true);
				}
				if (addNames) { //缁撴瀯鍖朼ddNames缇わ紝灏嗚鍔犲叆鐨勫姞鍏ュ埌oriName缇?					ar = addNames.split(_s._reClassToken);
					l = ar.length;
					for (i = 0; i < l; ++i) {
						(n = ar[i]) && !oriName[n] && (b = oriName[n] = true);
					}
				}
				if (removeNames) {
					ar = removeNames.split(_s._reClassToken);
					l = ar.length;
					for (i = 0; i < l; i++) {
						(n = ar[i]) && oriName[n] && (b = true) && delete oriName[n];
					}
				}
			}
			if (b) {
				ar.length = 0;
				for (var k in oriName) { //鏋勯€犵粨鏋滄暟缁?					ar.push(k);
				}
				oriName = ar.join(' ');
				elem.className = oriName;
			}
		}
		return oriName;
	},
	
	/**
	 * 鏌怘TMLElement鏄惁鍚湁鎸囧畾鐨勬牱寮忕被鍚嶇О
	 * @param {object} elem 鎸囧畾鐨凥TML鍏冪礌
	 * @param {string} name 鎸囧畾鐨勭被鍚嶇О
	 * @returns {boolean} 鏄惁鎿嶄綔鎴愬姛

	 * @example QZFL.css.hasClassName(document.getElementById("div_id"), "cname");
	 */
	hasClassName: function(elem, name){
		return (elem && name) ?
			(elem.classList ?
				elem.classList.contains(name)
					:
				(name && ((' ' + elem.className + ' ').indexOf(' ' + name + ' ') > -1))
			)
				:
			false;
	},
	
	/**
	 * 澧炲姞涓€缁勬牱寮忕被鍚?	 * @param {object} elem 鎸囧畾鐨凥TML鍏冪礌
	 * @param {string} names 鎸囧畾鐨勭被鍚嶇О
	 * @returns {string} 杩斿洖褰撳墠className

	 * @example QZFL.css.addClassName(document.getElementById("ele"), "cname imname");
	 */
	addClassName: function(elem, names){
		var _s = QZFL.css;

		return names && ((elem && elem.classList && !_s._reClassToken.test(names)) ? elem.classList.add(names) : _s.updateClassName(elem, null, names));
	},
	
	/**
	 * 闄ゅ幓涓€缁勬牱寮忕被鍚?	 * @param {object} elem 鎸囧畾鐨凥TML鍏冪礌
	 * @param {string} cname 鎸囧畾鐨勭被鍚嶇О
	 * @returns {string} 杩斿洖褰撳墠className

	 * @example QZFL.css.removeClassName($("ele"),"cname");
	 */
	removeClassName: function(elem, names){
		var _s = QZFL.css;

		return names && ((elem && elem.classList && !_s._reClassToken.test(names)) ? elem.classList.remove(names) : _s.updateClassName(elem, names));
	},
	
	/**
	 * 鏇挎崲涓ょ鏍峰紡绫诲悕
	 * @param {object|object[]} elems 鎸囧畾鐨凥TML鍏冪礌鎴栬€呬竴涓狧TML鍏冪礌闆嗗悎
	 * @param {string} a 鎸囧畾鐨勭被鍚嶇О
	 * @param {string} b 鎸囧畾鐨勭被鍚嶇О

	 * @example QZFL.css.replaceClassName($("ele"), "sourceClass", "targetClass");
	 */
	replaceClassName: function(elems, a, b){
		QZFL.css.swapClassName(elems, a, b, true);
	},
	
	/**
	 * 浜ゆ崲涓ょ鏍峰紡绫诲悕
	 * @param {object|object[]} elems 鎸囧畾鐨凥TML鍏冪礌鎴栬€呬竴涓狧TML鍏冪礌闆嗗悎
	 * @param {string} a 鎸囧畾鐨勭被鍚嶇О
	 * @param {string} b 鎸囧畾鐨勭被鍚嶇О
	 * @param {boolean} _isRep 鍙傛暟a,b鏄惁鍙嶅悜鍙浛鎹?
	 * @example QZFL.css.swapClassName($("div_id"), "classone", "classtwo", true);
	 */
	swapClassName: function(elems, a, b, _isRep){
		if (elems && typeof(elems) == "object") {
			if (elems.length === undefined) {
				elems = [elems];
			}
			for (var elem, i = 0, l = elems.length; i < l; ++i) {
				if ((elem = elems[i]) && elem.nodeType == 1) {
					if (QZFL.css.hasClassName(elem, a)) {
						QZFL.css.updateClassName(elem, a, b);
					} else if (!_isRep && QZFL.css.hasClassName(elem, b)) {
						QZFL.css.updateClassName(elem, b, a);
					}
				}
			}
		}
	},
	
	/**
	 * 寮€鍏虫牱寮忕被鍚嶏紝璋冧竴娆″姞涓婏紝鍐嶈皟涓€娆″共鎺夛紝鎴栧弽涔?	 * @param {object} elem 鎸囧畾鐨凥TML鍏冪礌
	 * @param {string} name 鎸囧畾鐨勭被鍚嶇О
	 * @returns {undefined}

	 * @example QZFL.css.toggleClassName($("ele"),"cname");
	 */
	toggleClassName: function(elem, name){
		if (!elem || elem.nodeType != 1) {
			return;
		}
		var _s = QZFL.css;
		if(elem.classList && name && !_s._reClassToken.test(name)){
			return elem.classList.toggle(name);
		}
		if (_s.hasClassName(elem, name)) {
			_s.updateClassName(elem, name);
		} else {
			_s.updateClassName(elem, null, name);
		}
	}	
};




/////////////
//dom.js
/////////////


/**
 * @fileoverview QZFL DOM 宸ュ叿闆嗭紝鍖呭惈瀵规祻瑙堝櫒DOM鐨勪竴浜涙搷浣? * @version $Rev: 1921 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $) - $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */


/**
 * QZFL dom 鎺ュ彛灏佽瀵硅薄銆傚娴忚鍣ㄥ父鐢ㄧ殑dom瀵硅薄鎺ュ彛杩涜娴忚鍣ㄥ吋瀹瑰皝瑁? *
 * @namespace QZFL dom 鎺ュ彛灏佽瀵硅薄
 */
QZFL.dom = {
	/**
	 * 鏍规嵁id鑾峰彇dom瀵硅薄
	 *
	 * @param {string} id 瀵硅薄ID
	 * @returns {object} 鎸囧畾id鐨凞OM鑺傜偣锛屾病鏈夋壘鍒颁负null
	 * @example QZFL.dom.getById("div_id");
	 */
	getById : function(id) {
		return document.getElementById(id);
	},

	/**
	 * 鏍规嵁name鑾峰彇dom闆嗗悎锛屾湁浜涙爣绛句緥濡俵i銆乻pan鏃犳硶閫氳繃getElementsByName鎷垮埌锛屽姞涓妕agName鎸囨槑灏卞彲浠?<br />
	 &lt;li name="n1">node1&lt;/li>&lt;span name="n1">node2&lt;/span>
	 * ie鍙兘鑾峰彇鍒發i锛岄潪ie涓嬩袱鑰呴兘鍙互
	 *
	 * @param {string} name 鎵€闇€鑺傜偣鐨刵ame
	 * @param {string} [tagName=""] 鏍囩鍚嶇ОtagName
	 * @param {object} [rt=undefined] 鏌ユ壘鐨勬牴瀵硅薄
	 * @returns {object[]} 鍖归厤鍒扮殑鑺傜偣闆嗗悎

	 * @example QZFL.dom.getByName("div_name");
	 */
	getByName : function(name, tagName, rt) {
		return QZFL.selector((tagName || "") + '[name="' + name + '"]', rt);
	},

	/**
	 * 鑾峰緱鍒跺畾鑺傜偣
	 *
	 * @param {string|object} e 鍖呮嫭id鍙凤紝鎴栧垯Html Element瀵硅薄
	 * @returns {object} 鍖归厤鍒扮殑鑺傜偣
	 * @example QZFL.dom.get("div_id");
	 */
	get : function(e) {
		return (typeof(e) == "string") ? document.getElementById(e) : e;
	},

	/**
	 * 鑾峰緱瀵硅薄
	 *
	 * @param {string|object} e 鍖呮嫭id鍙凤紝鎴栧垯HTML Node瀵硅薄
	 * @returns {object}
	 * @deprecated <strong style="color:red;">杩欎釜澶悶绗戜簡锛屼笉瑕佸啀鐢ㄤ簡</strong>
	 * @example QZFL.dom.getNode("div_id");
	 */
	getNode : function(e) {
		return (e && (e.nodeType || e.item)) ? e : document.getElementById(e);
	},
	/**
	 * 鍒犻櫎鑺傜偣
	 *
	 * @param {string|object} el HTML鍏冪礌鐨刬d鎴栬€匟TML鍏冪礌
	 * @example
QZFL.dom.removeElement("div_id");
QZFL.dom.removeElement(QZFL.dom.get("div_id2"));
	 */
	removeElement : function(elem) {
		if (elem = QZFL.dom.get(elem)) {
			if(QZFL.userAgent.ie > 8 && elem.tagName == "SCRIPT"){
				elem.src = "";
			}
			elem.removeNode ? elem.removeNode(true) : (elem.parentNode && elem.parentNode.removeChild(elem));
		}
		return elem = null;
	},

	/**
	 * 浠庝互鏌愬厓绱犲紑濮?瀵规寚瀹氬厓绱犲睘鎬х殑鍊间娇鐢ㄤ紶鍏ョ殑handler杩涜鍒ゆ柇锛宧andler杩斿洖true鏃舵煡璇㈠仠姝紝杩斿洖褰撳墠鍏冪礌<br />
	 鍚﹀垯浠ュ綋鍓嶅睘鎬у€兼墍鎸囩殑瀵硅薄涓烘牴锛岄€掑綊閲嶆柊鏌ユ壘锛屾渶缁堣繑鍥瀗ull
	 * @param {object} elem HTML鍏冪礌
	 * @param {string} prop 鏋勬垚閾剧殑鍏冪礌灞炴€у悕
	 * @param {function} func 妫€鏌ュ嚱鏁?杩斿洖true鐨勬椂鍊欏綋鍓嶆煡鎵剧粓缁? func = function(el){}; //浼犲叆褰撳墠鐨勮妭鐐筫l
	 * @returns {object} 缁撴灉瀵硅薄锛屾棤缁撴灉鏃惰繑鍥瀗ull

	 * @example
function getFirstChild(elem){ //鑾峰彇绗竴涓负HTMLElement鐨勫瓙鑺傜偣
	elem = QZFL.dom.get(elem);
	return QZFL.dom.searchChain(elem && elem.firstChild, 'nextSibling', function(el){
		return el.nodeType == 1;
	});
}
	 */
	searchChain : function(elem, prop, func){
		prop = prop || 'parentNode';
		while (elem && elem.nodeType && elem.nodeType == 1) {
			if (!func || func.call(elem, elem)) {
				return elem;
			}
			elem = elem[prop];
		}
		return null;
	},

	/**
	 * 閫氳繃褰撳墠鑺傜偣涓嶆柇寰€鐖剁骇鍚戜笂鏌ユ壘锛岀洿鍒版壘鍒板惈鏈夋寚瀹歝lassName鐨刣om鑺傜偣
	 *
	 * @param {string|object} el 瀵硅薄id鎴栧垯dom
	 * @param {string} className css绫诲悕
	 * @deprecated <strong>涓嶅缓璁娇鐢ㄤ簡锛岃浣跨敤 {@link QZFl.element}</strong>
	 * @returns {object} 绗竴涓粨鏋滆妭鐐?	 */
	searchElementByClassName : function(elem, className){
		elem = QZFL.dom.get(elem);
		return QZFL.dom.searchChain(elem, 'parentNode', function(el){
			return QZFL.css.hasClassName(el, className);
		});
	},
	/**
	 * 鑾峰彇鎸囧畾className鐨勬墍鏈夊瓙鑺傜偣
	 *
	 * @param {string} className 鎸囧畾鐨刢lass鍊?	 * @param {string} [tagName] 鑺傜偣鍚?	 * @param {string|object} context 鍙兘鐨勬牴瀵硅薄
	 * @deprecated <strong>涓嶅缓璁娇鐢ㄤ簡锛岃浣跨敤 {@link QZFl.element}</strong>
	 * @returns {object[]} 缁撴灉鑺傜偣闆嗗悎
	 */
	getElementsByClassName : function(className, tagName, context) {
		return QZFL.selector((tagName || '') + '.' + className, QZFL.dom.get(context));
	},


	/**
	 * 鍒ゆ柇鎸囧畾鐨勮妭鐐规槸鍚︽槸绗簩涓妭鐐圭殑绁栧厛
	 *
	 * @param {object} a 瀵硅薄锛岀埗鑺傜偣
	 * @param {object} b 瀵硅薄锛屽瓙瀛欒妭鐐?	 * @returns {boolean} true鍗砨鏄痑鐨勫瓙鑺傜偣锛屽惁鍒欎负false
	 * @example  QZFL.dom.isAncestor(QZFL.dom.get("div1"), QZFL.dom.get("div2"))
	 */
	isAncestor : function(a, b) {
		return a && b && a != b && QZFL.dom.contains(a, b);
	},



	/**
	 * 鏍规嵁鍑芥暟寰楀埌鐗瑰畾鐨勭埗鑺傜偣
	 *
	 * @param {object|string} node瀵硅薄鎴栧叾id
	 * @param {string} method 鍒涘缓瀵硅薄鐨凾agName
	 * @returns {object}
	 * @example var node = QZFL.dom.getAncestorBy($("div_id"), "div");
	 */
	getAncestorBy : function(elem, method){
		elem = QZFL.dom.get(elem);
		return QZFL.dom.searchChain(elem.parentNode, 'parentNode', function(el){
			return el.nodeType == 1 && (!method || method(el));
		});
	},


	/**
	 * 寰楀埌绗竴涓狧TMLElement瀛愯妭鐐?	 *
	 * @param {object|string} HTMLElement瀵硅薄鎴栧叾id
	 * @returns {object} 缁撴灉瀵硅薄
	 * @example var element = QZFL.dom.getFirstChild("el_id");
	 */
	getFirstChild : function(elem){
		elem = QZFL.dom.get(elem);
		return elem.firstElementChild || QZFL.dom.searchChain(
			elem && elem.firstChild,
			'nextSibling',
			function(el){
				return el.nodeType == 1;
			}
		);
	},

	/**
	 * 寰楀埌鏈€鍚庝竴涓瓙HTMLElement鑺傜偣
	 *
	 * @param {object|string} node瀵硅薄鎴栧叾id
	 * @returns {object}
	 * @example var element = QZFL.dom.getFirstChild(QZFL.dom.get("el_id"));
	 */
	getLastChild : function(elem){
		elem = QZFL.dom.get(elem);
		return elem.lastElementChild || QZFL.dom.searchChain(
			elem && elem.lastChild,
			'previousSibling',
			function(el){
				return el.nodeType == 1;
			}
		);
	},


	/**
	 * 寰楀埌涓嬩竴涓厔HTMLElement寮熻妭
	 *
	 * @param {string|object} node瀵硅薄鎴栧叾id
	 * @returns 锝沷bject}
	 * @example QZFL.dom.getNextSibling("el_id");
	 */
	getNextSibling : function(elem){
		elem = QZFL.dom.get(elem);
		return elem.nextElementSibling || QZFL.dom.searchChain(
			elem && elem.nextSibling,
			'nextSibling',
			function(el){
				return el.nodeType == 1;
			}
		);
	},
	/**
	 * 寰楀埌涓婁竴涓厔寮烪TMLElement鑺傜偣
	 *
	 * @param {object|string} node瀵硅薄鎴栧叾id
	 * @returns {object}
	 * @example QZFL.dom.getPreviousSibling(QZFL.dom.get("el_id"));
	 */
	getPreviousSibling : function(elem){
		elem = QZFL.dom.get(elem);
		return elem.previousElementSibling || QZFL.dom.searchChain(
			elem && elem.previousSibling,
			'previousSibling',
			function(el){
				return el.nodeType == 1;
			}
		);
	},


	/**
	 * 浜ゆ崲涓や釜鑺傜偣
	 *
	 * @param {object} node1 node瀵硅薄
	 * @param {object} node2 node瀵硅薄

	 * @example QZFL.dom.swapNode(QZFL.dom.get("el_one"),QZFL.dom.get("el_two"))
	 */
	swapNode : function(node1, node2) {
		// for ie
		if (node1.swapNode) {
			node1.swapNode(node2);
		} else {
			var prt = node2.parentNode,
				next = node2.nextSibling;

			if (next == node1) {
				prt.insertBefore(node1, node2);
			} else if (node2 == node1.nextSibling) {
				prt.insertBefore(node2, node1);
			} else {
				node1.parentNode.replaceChild(node2, node1);
				prt.insertBefore(node1, next);
			}
		}
	},


	/**
	 * 瀹氱偣鍒涘缓Dom瀵硅薄锛屼竴鍙ヨ瘽鎶婁竴涓妭鐐瑰垱寤哄湪鍙︿竴涓妭鐐瑰唴锛屾湁鐐规噿
	 *
	 * @param {string} [tagName='div'] 鍒涘缓瀵硅薄鐨凾agName
	 * @param {string|object} [elem=document.body] 瀹瑰櫒瀵硅薄id鎴栧垯dom
	 * @param {boolean} [insertFirst=false] 鏄惁浠庡墠閮ㄦ彃鍏?	 * @param {object} [attrs=undefined] 瀵硅薄灞炴€у垪琛紝渚嬪 {id:"newDom1",style:"color:#000"}

	 * @example QZFL.dom.createElementIn("div",document.body,false,{id:"newDom1",style:"color:#000"})
	 * @returns {object} 杩斿洖鍒涘缓濂界殑dom寮曠敤
	 */
	createElementIn : function(tagName, elem, insertFirst, attrs){
		var _e = (elem = QZFL.dom.get(elem) || document.body).ownerDocument.createElement(tagName || "div"), k;
		
		// 璁剧疆Element灞炴€?		if (typeof(attrs) == 'object') {
			for (k in attrs) {
				if (k == "class") {
					_e.className = attrs[k];
				} else if (k == "style") {
					_e.style.cssText = attrs[k];
				} else {
					_e[k] = attrs[k];
				}
			}
		}
		insertFirst ? elem.insertBefore(_e, elem.firstChild) : elem.appendChild(_e);
		return _e;
	},


	/**
	 * 鑾峰彇瀵硅薄娓叉煋鍚庣殑鏍峰紡瑙勫垯
	 *
	 * @param {string|object} el 瀵硅薄id鎴栧垯dom
	 * @param {string} property 鏍峰紡瑙勫垯鍚嶏紝璇蜂娇鐢╦s璇硶锛屽z-index瀵瑰簲鐨勬槸zIndex
	 * @example var width=QZFL.dom.getStyle("div_id","width");//width=163px;
	 * @returns {string} 鏍峰紡鍊?	 */
	getStyle : function(el, property) {
		el = QZFL.dom.get(el);

		if (!el || el.nodeType == 9) {
			return null;
		}

		var w3cMode = document.defaultView && document.defaultView.getComputedStyle,
			computed = !w3cMode ? null : document.defaultView.getComputedStyle(el, ''),
			value = "";

		switch (property) {
			case "float" :
				property = w3cMode ? "cssFloat" : "styleFloat";
				break;
			case "opacity" :
				if (!w3cMode) { // IE Mode
					var val = 100;
					try {
						val = el.filters['DXImageTransform.Microsoft.Alpha'].opacity;
					} catch (e) {
						try {
							val = el.filters('alpha').opacity;
						} catch (e) {}
					}
					return val / 100;
				}else{
					return parseFloat((computed || el.style)[property]);
				}
				break;
			case "backgroundPositionX" : // 鍙湁ie鍜寃ebkit娴忚鍣ㄦ敮鎸?				// background-position-x
				if (w3cMode) {
					property = "backgroundPosition";
					return ((computed || el.style)[property]).split(" ")[0];
				}
				break;
			case "backgroundPositionY" : // 鍙湁ie鍜寃ebkit娴忚鍣ㄦ敮鎸?				// background-position-y
				if (w3cMode) {
					property = "backgroundPosition";
					return ((computed || el.style)[property]).split(" ")[1];
				}
				break;
		}

		if (w3cMode) {
			return (computed || el.style)[property];
		} else {
			return (el.currentStyle[property] || el.style[property]);
		}
	},

	/**
	 * 璁剧疆鏍峰紡瑙勫垯
	 *
	 * @param {string|object} el 瀵硅薄id鎴栧垯dom
	 * @param {string|object} properties 鏍峰紡瑙勫垯琛紝濡倇zIndex:10000, height:"200px"}锛屾垨鍗曚釜瀛楃涓叉牱寮忓悕
	 * @param {string|number} [value] 鑻ヤ笂涓€涓弬鏁版槸瀛楃涓插舰寮忕殑鏍峰紡鍚嶏紝杩欓噷缁欏嚭鏍峰紡鍊?	 * @example <pre>
QZFL.dom.setStyle("div_id", "width", "200px");
QZFL.dom.setStyle("div_id", {"width" : "200px", "height" : "300px"});</pre>

	 * @returns {boolean} 鎴愬姛杩斿洖true
	 */
	setStyle : function(el, properties, value) {
	    var DOM = QZFL.dom;
		if (!(el = DOM.get(el)) || el.nodeType != 1) {
			return false;
		}

		var tmp, bRtn = true, re;
		if (typeof(properties) == 'string') {
			tmp = properties;
			properties = {};
			properties[tmp] = value;
		}
		
		for (var prop in properties) {
			value = properties[prop];
			re = DOM.convertStyle(el, prop, value);
			prop = re.prop; value = re.value;
			if (typeof el.style[prop] != "undefined") {
				el.style[prop] = value;
				bRtn = bRtn && true;
			} else {
				bRtn = bRtn && false;
			}
		}
		return bRtn;
	},
	/**
	 * 杞崲灞炴€у拰鍊?	 */
    convertStyle : function(el, prop, value){
        var DOM = QZFL.dom, tmp, rexclude = /z-?index|font-?weight|opacity|zoom|line-?height/i, w3cMode;
        w3cMode = ((tmp = document.defaultView) && tmp.getComputedStyle);
        if (prop == 'float') {
            prop = w3cMode ? "cssFloat" : "styleFloat";
        } else if (prop == 'opacity') {
            if (!w3cMode) { // for ie only
                prop = 'filter';
                value = value >= 1 ? '' : ('alpha(opacity=' + Math.round(value * 100) + ')');
            }
        } else if (prop == 'backgroundPositionX' || prop == 'backgroundPositionY') {
            tmp = prop.slice(-1) == 'X' ? 'Y' : 'X';
            if (w3cMode) {
                var v = QZFL.dom.getStyle(el, "backgroundPosition" + tmp);
                prop = 'backgroundPosition';
                typeof(value) == 'number' && (value = value + 'px');
                value = tmp == 'Y' ? (value + " " + (v || "top")) : ((v || 'left') + " " + value);
            }
        }
        value += (typeof value === "number" && !rexclude.test(prop) ? 'px' : '');
        return {
            'prop' : prop,
            'value' : value
        };
	
    },
    

	/**
	 * 寤虹珛鏈塶ame灞炴€х殑element
	 * 
	 * @param {string} type node鐨則agName
	 * @param {string} name name灞炴€у€?	 * @param {object} [doc=document] 鎵€鍦ㄦ枃妗ｆ爲
	 * @returns {object} 缁撴灉element
	 * @example QZFL.dom.createNamedElement("div","div_name",QZFL.dom.get("doc"));
	 */
	createNamedElement : function(type, name, doc) {
		var _doc = doc || document,
			element;

		try {
			element = _doc.createElement('<' + type + ' name="' + name + '">');
		} catch (ign) {}

		if (!element) {
			element = _doc.createElement(type);
		}

		if (!element.name) {
			element.name = name;
		}
		return element;
	},

	/**
	 * 鑾峰彇鑺傜偣鐭╁舰璇存槑绗?	 * @param {object} elem
	 * @returns {object} 杩斿洖浣嶇疆璇存槑瀵硅薄 {"top","left","width","height"}
	 */
	getRect : function(elem){
		if (elem = QZFL.dom.get(elem)) {
			var box = QZFL.object.extend({}, elem.getBoundingClientRect());

			if (typeof box.width == 'undefined') {
				box.width = box.right - box.left;
				box.height = box.bottom - box.top;
			}
			return box;
		}
	},

	/**
	 * 鑾峰彇瀵硅薄鍧愭爣鍜屽昂瀵?	 *
	 * @param {object} elem
	 * @returns {object} 杩斿洖浣嶇疆璇存槑瀵硅薄 {"top","left","width","height"};
	 * @example var position = QZFL.dom.getPosition(QZFL.dom.get("div_id"));
	 */
	getPosition : function(elem){
		var box, s, doc;
		if (box = QZFL.dom.getRect(elem)) {
			if (s = QZFL.dom.getScrollLeft(doc = elem.ownerDocument)) {
				box.left += s, box.right += s;
			}
			if (s = QZFL.dom.getScrollTop(doc)) {
				box.top += s, box.bottom += s;
			}
			return box;
		}
	},

	/**
	 * 璁剧疆瀵硅薄鍧愭爣鍜屽昂瀵?	 *
	 * @param {object} el
	 * @param {object} pos 浣嶇疆鍜屽ぇ灏忔弿杩板璞?	 * @example QZFL.dom.setPosition(QZFL.dom.get("div_id"),{"100px","100px","400px","300px"});
	 */
	setPosition : function(el, pos) {
		QZFL.dom.setXY(el, pos['left'], pos['top']);
		QZFL.dom.setSize(el, pos['width'], pos['height']);
	},

	/**
	 * 鑾峰彇瀵硅薄鍧愭爣
	 *
	 * @param {object} elem
	 * @returns {object} 鏁扮粍缁撴瀯 [top鍊? left鍊糫
	 * @example var xy=QZFL.dom.getXY(QZFL.dom.get("div_id"));
	 */
	getXY : function(elem){
		var box = QZFL.dom.getPosition(elem) || { left: 0, top: 0 };
		return [box.left, box.top];
	},

	/**
	 * 鑾峰彇瀵硅薄灏哄
	 *
	 * @param {object} elem
	 * @returns {object} [width鍊? height鍊糫
	 * @example var size = QZFL.dom.getSize(QZFL.dom.get("div_id"));
	 */
	getSize : function(elem){
		var box = QZFL.dom.getPosition(elem) || { width: -1, height: -1 };
		return [box.width, box.height];
	},

	/**
	 * 璁剧疆dom鍧愭爣
	 *
	 * @param {object} elem
	 * @param {string|number} x 妯潗鏍?	 * @param {string|number} y 绾靛潗鏍?	 * @example QZFL.dom.setXY(QZFL.dom.get("div_id"),400,200);
	 */
	setXY : function(elem, x, y){
		var _ml = parseInt(QZFL.dom.getStyle(elem, "marginLeft")) || 0,
			_mt = parseInt(QZFL.dom.getStyle(elem, "marginTop")) || 0;

		QZFL.dom.setStyle(elem, {
			left: ((parseInt(x, 10) || 0) - _ml) + "px",
			top: ((parseInt(y, 10) || 0) - _mt) + "px"
		});
	},

	/**
	 * 鑾峰彇瀵硅薄scrollLeft鐨勫€?	 *
	 * @param {object} [doc=document] 鎵€闇€妫€鏌ョ殑椤甸潰document寮曠敤
	 * @returns {number}
	 * @example QZFL.dom.getScrollLeft(document);
	 */
	getScrollLeft : function(doc) {
		var _doc = doc || document;
		return (_doc.defaultView && _doc.defaultView.pageXOffset) || Math.max(_doc.documentElement.scrollLeft, _doc.body.scrollLeft);
	},

	/**
	 * 鑾峰彇瀵硅薄鐨剆crollTop鐨勫€?	 *
	 * @param {object} [doc=document] 鎵€闇€妫€鏌ョ殑椤甸潰document寮曠敤
	 * @returns {number}
	 * @example QZFL.dom.getScrollTop(document);
	 */
	getScrollTop : function(doc) {
		var _doc = doc || document;
		return (_doc.defaultView && _doc.defaultView.pageYOffset) || Math.max(_doc.documentElement.scrollTop, _doc.body.scrollTop);
	},

	/**
	 * 鑾峰彇瀵硅薄scrollHeight鐨勫€?	 *
	 * @param {object} [doc=document] 鎵€闇€妫€鏌ョ殑椤甸潰document寮曠敤
	 * @returns {number}
	 * @example QZFL.dom.getScrollHeight(document);
	 */
	getScrollHeight : function(doc) {
		var _doc = doc || document;
		return Math.max(_doc.documentElement.scrollHeight, _doc.body.scrollHeight);
	},

	/**
	 * 鑾峰彇瀵硅薄鐨剆crollWidth鐨勫€?	 *
	 * @param {object} [doc=document] 鎵€闇€妫€鏌ョ殑椤甸潰document寮曠敤
	 * @returns {number}
	 * @example QZFL.dom.getScrollWidht(document);
	 */
	getScrollWidth : function(doc) {
		var _doc = doc || document;
		return Math.max(_doc.documentElement.scrollWidth, _doc.body.scrollWidth);
	},

	/**
	 * 璁剧疆瀵硅薄scrollLeft鐨勫€?	 *
	 * @param {number} value scroll left鐨勪慨鏀瑰€?	 * @param {object} [doc=document] 鎵€闇€妫€鏌ョ殑椤甸潰document寮曠敤
	 * @example QZFL.dom.setScrollLeft(200,document);
	 */
	setScrollLeft : function(value, doc) {
		var _doc = doc || document;
		_doc[_doc.compatMode == "CSS1Compat" && !QZFL.userAgent.webkit ? "documentElement" : "body"].scrollLeft = value;
	},

	/**
	 * 璁剧疆瀵硅薄鐨剆crollTop鐨勫€?	 *
	 * @param {number} value scroll top鐨勪慨鏀瑰€?	 * @param {object} [doc=document] 鎵€闇€妫€鏌ョ殑椤甸潰document寮曠敤
	 * @example QZFL.dom.setScrollTop(200,document);
	 */
	setScrollTop : function(value, doc) {
		var _doc = doc || document;
		_doc[_doc.compatMode == "CSS1Compat" && !QZFL.userAgent.webkit ? "documentElement" : "body"].scrollTop = value;
	},

	/**
	 * 鑾峰彇瀵硅薄鐨勫彲瑙嗗尯鍩熼珮搴?	 *
	 * @param {object} [doc=document] 鎵€闇€妫€鏌ョ殑椤甸潰document寮曠敤
	 * @returns {number}
	 * @example QZFL.dom.getClientHeight();
	 */
	getClientHeight : function(doc) {
		var _doc = doc || document;
		return _doc.compatMode == "CSS1Compat" ? _doc.documentElement.clientHeight : _doc.body.clientHeight;
	},

	/**
	 * 鑾峰彇瀵硅薄鐨勫彲瑙嗗尯鍩熷搴?	 *
	 * @param {object} [doc=document] 鎵€闇€妫€鏌ョ殑椤甸潰document寮曠敤
	 * @returns {number}
	 * @example QZFL.dom.getClientWidth();
	 */
	getClientWidth : function(doc) {
		var _doc = doc || document;
		return _doc.compatMode == "CSS1Compat" ? _doc.documentElement.clientWidth : _doc.body.clientWidth;
	},
	

	/**
	 * size鏁板€奸渶瑕佺敤鐨勬ā寮?	 * @private
	 *
	 */
	_SET_SIZE_RE : /^\d+(?:\.\d*)?(px|%|em|in|cm|mm|pc|pt)?$/,

	/**
	 * 璁剧疆dom灏哄
	 *
	 * @param {string|object} el 鑺傜偣ID鎴栬€呰妭鐐规湰韬紩鐢?	 * @param {string|number} width 瀹藉害
	 * @param {string|number} height 楂樺害
	 * @example QZFL.dom.setSize($("abc"), 100, 200);
	 */
	setSize : function(el, w, h){
		el = QZFL.dom.get(el);
		var _r = QZFL.dom._SET_SIZE_RE,
			m;

		QZFL.dom.setStyle(el, "width", (m=_r.exec(w)) ? (m[1] ? w : (parseInt(w,10)+'px')) : 'auto');
		QZFL.dom.setStyle(el, "height",(m=_r.exec(h)) ? (m[1] ? h : (parseInt(h,10)+'px')) : 'auto');
	},


	/**
	 * 鑾峰彇document鐨剋indow瀵硅薄
	 *
	 * @param {object} [doc=document] 鎵€闇€妫€鏌ョ殑椤甸潰document寮曠敤
	 * @returns {object} 杩斿洖缁撴灉window瀵硅薄
	 * @example QZFL.dom.getDocumentWindow();
	 */
	getDocumentWindow : function(doc) {
		var _doc = doc || document;
		return _doc.parentWindow || _doc.defaultView;
	},

	/**
	 * 鎸塗agname鑾峰彇鎸囧畾鍛藉悕绌洪棿鐨勮妭鐐?	 *
	 * @param {object} [node=document] 鎵€闇€閬嶅巻鐨勬牴鑺傜偣
	 * @param {string} ns 鍛藉悕绌洪棿鍚?	 * @param {string} tgn 鏍囩鍚?	 * @returns {object} 缁撴灉鏁扮粍
	 * @example QZFL.dom.getElementsByTagNameNS(document, "qz", "div");
	 */
	getElementsByTagNameNS : function(node, ns, tgn) {
		node = node || document;
		var res = [];

		if (node.getElementsByTagNameNS) {
			return node.getElementsByTagName(ns + ":" + tgn);
		} else if (node.getElementsByTagName) {
			var n = document.namespaces;
			if (n.length > 0) {
				var l = node.getElementsByTagName(tgn);
				for (var i = 0, len = l.length; i < len; ++i) {
					if (l[i].scopeName == ns) {
						res.push(l[i]);
					}
				}
			}
		}

		return res;
	},


	/**
	 * 浠庝竴涓粰鍑鸿妭鐐瑰悜涓婂鎵句竴涓猼agName鐩哥鐨勮妭鐐?	 *
	 * @param {object} elem 缁欏嚭鐨勮妭鐐?	 * @param {string} tn 闇€瑕佹煡鎵剧殑鑺傜偣tag name
	 * @returns {object} 缁撴灉锛屾病鎵惧埌鏄痭ull
	 * @example QZFL.dom.getElementByTagNameBubble(QZFL.dom.get("div_id"),"div");
	 */
	getElementByTagNameBubble : function(elem, tn){
		if(!tn){
			return null;
		}
		var maxLv = 15;
		tn = String(tn).toUpperCase();
		if(tn == 'BODY'){
			return document.body;
		}
		elem = QZFL.dom.searchChain(
			elem = QZFL.dom.get(elem),
			'parentNode',
			function(el){
				return el.tagName == tn || el.tagName == 'BODY' || (--maxLv) < 0;
			}
		);
		return !elem || maxLv < 0 ? null : elem;
	},

	/**
	 * 鍦ㄥ厓绱犵浉閭荤殑浣嶇疆(鍏蜂綋浣嶇疆鍙€?鎻掑叆 html鏂囨湰  text绾枃鏈? element鑺傜偣
	 * @param {object} elem 鍏冪礌寮曠敤
	 * @param {number} where 鍙栧€? 1 2 3锛屽垎鍒搴旓細beforeBegin, afterBegin, beforeEnd, afterEnd
	 * @param {object|string} html html鏂囨湰 鎴?text鏅€氭枃鏈?鎴?element鑺傜偣寮曠敤
	 * @param {boolean} [isText=false] 褰撻渶瑕佹彃鍏ext鏃讹紝鐢ㄦ鍙傛暟鍖哄埆浜巋tml
	 * @returns {boolean} 鎿嶄綔鏄惁鎴愬姛
	 * @example
QZFL.dom.insertAdjacent($("test"), 1, "world!", true); //0 1 2 3 鍒嗗埆浠ｈ〃锛氳妭鐐瑰鑺傜偣鍓嶏紱鑺傜偣鍐呭ご閮紱鑺傜偣鍐呭熬閮紱鑺傜偣澶栬妭鐐瑰悗
	 */
	insertAdjacent : function(elem, where, html, isText){
		var range,
			pos = ['beforeBegin', 'afterBegin', 'beforeEnd', 'afterEnd'],
			doc;

		if (QZFL.lang.isElement(elem) && pos[where] && (QZFL.lang.isString(html) || QZFL.lang.isElement(html))) {
			if (elem.insertAdjacentHTML && elem.insertAdjacentElement && elem.insertAdjacentText) {
				elem['insertAdjacent' + (typeof(html) == 'object' ? 'Element' : (isText ? 'Text' : 'HTML'))](pos[where], html);
			} else {
				range = (doc = elem.ownerDocument).createRange();
				range[where == 1 || where == 2 ? 'selectNodeContents' : 'selectNode'](elem);
				range.collapse(where < 2);
				range.insertNode(typeof(html) != 'string' ? html : isText ? doc.createTextNode(html) : range.createContextualFragment(html));
			}
			return true;
		}
		return false;
	}
};





/////////////
//event.js
/////////////


/**
 * @fileoverview QZFL 浜嬩欢椹卞姩鍣紝缁欐祻瑙堝櫒鎻愪緵鍩烘湰鐨勪簨浠堕┍鍔ㄦ帴鍙? * @version 1.$Rev: 1921 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */

/**
 * 浜嬩欢椹卞姩瀵硅薄锛屽寘鍚澶氫簨浠堕┍鍔ㄤ互鍙婄粦瀹氱瓑鏂规硶,鍏抽敭
 *
 * @namespace QZFL 浜嬩欢椹卞姩鍣紝缁欐祻瑙堝櫒鎻愪緵鍩烘湰鐨勪簨浠堕┍鍔ㄦ帴鍙? */
QZFL.event = {
	/**
	 * 鎸夐敭浠ｇ爜鏄犲皠
	 *
	 * @namespace QZFL.event.KEYS 閲岄潰鍖呭惈浜嗗鎸夐敭鐨勬槧灏?	 * @type Object
	 */
	KEYS : {
		/**
		 * 閫€鏍奸敭
		 */
		BACKSPACE : 8,
		/**
		 * tab
		 */
		TAB : 9,
		RETURN : 13,
		ESC : 27,
		SPACE : 32,
		LEFT : 37,
		UP : 38,
		RIGHT : 39,
		DOWN : 40,
		DELETE : 46
	},
	//杩欎釜涓滀笢涓嶉渶瑕佷簡鍚?	/**
	 * 鎵╁睍绫诲瀷锛岃繖绫讳簨浠跺湪缁戝畾鐨勬椂鍊欏厑璁镐紶鍙傛暟锛屽苟涓旂敤鏉ョ壒娈婂鐞嗕竴浜涚壒鍒殑浜嬩欢缁戝畾
	 *
	 * @ignore
	 *
	extendType : /(click|mousedown|mouseover|mouseout|mouseup|mousemove|scroll|contextmenu|resize)/i,*/


	/**
	 * 鍏ㄥ眬浜嬩欢鏍?	 * @ignore
	 */
	_eventListDictionary : {},

	/**
	 * @ignore
	 */
	_fnSeqUID : 0,

	/**
	 * @ignore
	 */
	_objSeqUID : 0,

	/**
	 * 浜嬩欢缁戝畾
	 *
	 * @param {DocumentElement} obj 闇€瑕佹坊鍔犱簨浠剁殑椤甸潰瀵硅薄
	 * @param {String} eventType 闇€瑕佹坊鍔犵殑浜嬩欢
	 * @param {Function} fn 浜嬩欢闇€瑕佺粦瀹氬埌鐨勫鐞嗗嚱鏁?	 * @param {Array} argArray 鍙傛暟鏁扮粍
	 * @type Boolean
	 * @version 1.1 memory leak optimise by scorr
	 * @author zishunchen
	 * @return 鏄惁缁戝畾鎴愬姛(true涓烘垚鍔燂紝false涓哄け璐?
	 * @example QZFL.event.addEvent(QZFL.dom.get('demo'),'click',hello);
	 */
	addEvent : function(obj, eventType, fn, argArray) {
		var cfn,
			res = false,
			l,
			handlers,
			efn,
			sTime;

		if (!obj) {
			return res;
		}
		if (!obj.eventsListUID) {
			obj.eventsListUID = "e" + (++QZFL.event._objSeqUID);
		}

		if (!(l = QZFL.event._eventListDictionary[obj.eventsListUID])) {
			l = QZFL.event._eventListDictionary[obj.eventsListUID] = {};
		}

		if (!fn.__elUID) {
			fn.__elUID = "e" + (++QZFL.event._fnSeqUID) + obj.eventsListUID;
		}

		//鎻掑叆ipad璁惧涓妋ouseover鍜宮ouseout鐨勫吋瀹?		if(QZFL.userAgent.isiPad && ((eventType == 'mouseover') || (eventType == 'mouseout'))){
			cfn = function(evt){
				sTime = new Date().getTime();
			}
			l['_'+eventType] = fn;
			if(l._ipadBind){
				return false;
			}
			eventType = 'touchstart';
			l._ipadBind = 1;
			
			efn = function(evt){
				var t = new Date().getTime() - sTime,fn;
				if(t < 700){//mouseover鎴栬€卭ut
					fn = l._mouseover;
					if(l._ismouseover){
						fn = l._mouseout;
						l._ismouseover = 0
					}else{
						l._ismouseover = 1;
					}
					QZFL.event.preventDefault(evt);
					return fn && fn.apply(obj, !argArray ? [QZFL.event.getEvent(evt)] : ([QZFL.event.getEvent(evt)]).concat(argArray));
				}//鍏朵粬鐨勪负click
				return true;
			}
			QZFL.event.addEvent(obj, 'touchend', efn);
		}
		//鍏煎end

		if (!l[eventType]) {
			l[eventType] = {};
		}
		
		if (!l[eventType].handlers) {
            l[eventType].handlers = {};
        }
        handlers = l[eventType].handlers;

		if(typeof(handlers[fn.__elUID]) == 'function'){
			return false;
		}

		cfn = cfn || function(evt) {
				return fn.apply(obj, !argArray ? [QZFL.event.getEvent(evt)] : ([QZFL.event.getEvent(evt)]).concat(argArray));
			};

		if (obj.addEventListener) {
			obj.addEventListener(eventType, cfn, false);
			res = true;
		} else if (obj.attachEvent) {
			res = obj.attachEvent("on" + eventType, cfn);
		} else {
			res = false;
		}
		if (res) {
			handlers[fn.__elUID] = cfn;
		}
		return res;
	},
	/**
	 * 鎵嬪姩瑙﹀彂鍥炶皟
	 * @param {HTMLElement} obj 瑙﹀彂鐨勮妭鐐?	 * @param {String} eventType 浜嬩欢绫诲瀷
	 */
	trigger : function(obj, eventType){
		var l = obj && QZFL.event._eventListDictionary[obj.eventsListUID],
			handlers = l && l[eventType] && l[eventType].handlers, i;
		if(handlers){
			try{
				for(i in handlers){
					handlers[i].call(window, {});
				}
			}catch(evt){QZFL.console.print('QZFL.event.trigger error')}
		}
	},
	/**
	 * 鏂规硶鍙栨秷缁戝畾
	 *
	 * @param {DocumentElement} obj 闇€瑕佸彇娑堜簨浠剁粦瀹氱殑椤甸潰瀵硅薄
	 * @param {String} eventType 闇€瑕佸彇娑堢粦瀹氱殑浜嬩欢
	 * @param {Function} fn 闇€瑕佸彇娑堢粦瀹氱殑鍑芥暟
	 * @return 鏄惁鎴愬姛鍙栨秷(true涓烘垚鍔燂紝false涓哄け璐?
	 * @type Boolean
	 * @version 1.1 memory leak optimise by scorr
	 * @author zishunchen
	 * @example QZFL.event.removeEvent(QZFL.dom.get('demo'),'click',hello);
	 */
	removeEvent : function(obj, eventType, fn) {
		var cfn = fn,
			res = false,
			l = QZFL.event._eventListDictionary,
			r;

		if (!obj) {
			return res;
		}
		if (!fn) {
			return QZFL.event.purgeEvent(obj, eventType);
		}

		if (obj.eventsListUID && l[obj.eventsListUID] && l[obj.eventsListUID][eventType]) {
			l = l[obj.eventsListUID][eventType].handlers;
			if(l && l[fn.__elUID]){
				cfn = l[fn.__elUID];
				r = l;
			}
		}

		if (obj.removeEventListener) {
			obj.removeEventListener(eventType, cfn, false);
			res = true;
		} else if (obj.detachEvent) {
			obj.detachEvent("on" + eventType, cfn);
			res = true;
		} else {
			//rt.error("Error.!.");
			return false;
		}
		if (res && r && r[fn.__elUID]) {
			delete r[fn.__elUID];
		}
		return res;
	},

	/**
	 * 鍙栨秷鍏ㄩ儴鏌愮被鍨嬬殑鏂规硶缁戝畾
	 *
	 * @param {DocumentElement} obj 闇€瑕佸彇娑堜簨浠剁粦瀹氱殑椤甸潰瀵硅薄
	 * @param {String} eventType 闇€瑕佸彇娑堢粦瀹氱殑浜嬩欢
	 * @example QZFL.event.purgeEvent(QZFL.dom.get('demo'),'click');
	 * @return {Boolean} 鏄惁鎴愬姛鍙栨秷(true涓烘垚鍔燂紝false涓哄け璐?
	 */
	purgeEvent : function(obj, type) {
		var l, h;
		if (obj.eventsListUID && (l = QZFL.event._eventListDictionary[obj.eventsListUID]) && l[type] && (h = l[type].handlers)) {
			for (var k in h) {
				if (obj.removeEventListener) {
					obj.removeEventListener(type, h[k], false);
				} else if (obj.detachEvent) {
					obj.detachEvent('on' + type, h[k]);
				}
			}
		}
		if (obj['on' + type]) {
			obj['on' + type] = null;
		}
		if (h) {
			l[type].handlers = null;
			delete l[type].handlers;
		}
		return true;
	},

	/**
	 * 鏍规嵁涓嶅悓娴忚鍣ㄨ幏鍙栧搴旂殑Event瀵硅薄
	 *
	 * @param {Event} evt
	 * @return 淇杩囩殑Event瀵硅薄, 鍚屾椂杩斿洖涓€涓慨姝utton鐨勮嚜瀹氫箟灞炴€?
	 * @type Event
	 * @example QZFL.event.getEvent();
	 * @return Event
	 */
	getEvent: function(evt) {
		var evt = window.event || evt || null,
			c, _s = QZFL.event.getEvent, ct = 0;

		if(!evt){
			c = arguments.callee;

			while(c && ct < _s.MAX_LEVEL){
				if(c.arguments && (evt = c.arguments[0]) && (typeof(evt.button) != "undefined" && typeof(evt.ctrlKey) != "undefined")){
					break;
				}
				++ct;
				c = c.caller;
			}
		}
		return evt;
	},

	/**
	 * 鑾峰緱榧犳爣鎸夐敭
	 *
	 * @param {Object} evt
	 * @example QZFL.event.getButton(evt);
	 * @return {number} 榧犳爣鎸夐敭 -1=鏃犳硶鑾峰彇event 0=宸﹂敭 1= 涓敭 2= 鍙抽敭
	 */
	getButton : function(evt) {
		var e = QZFL.event.getEvent(evt);
		if (!e) {
			return -1
		}

		if (QZFL.userAgent.ie) {
			return e.button - Math.ceil(e.button / 2);
		} else {
			return e.button;
		}
	},

	/**
	 * 杩斿洖浜嬩欢瑙﹀彂鐨勫璞?	 *
	 * @param {Object} evt
	 * @example QZFL.event.getTarget(evt);
	 * @return {object}
	 */
	getTarget : function(evt) {
		var e = QZFL.event.getEvent(evt);
		if (e) {
			return e.srcElement || e.target;
		} else {
			return null;
		}
	},

	/**
	 * 杩斿洖鑾峰緱鐒︾偣鐨勫璞?	 *
	 * @param {Object} evt
	 * @example QZFL.event.getCurrentTarget();
	 * @return {object}
	 */
	getCurrentTarget : function(evt) {
		var e = QZFL.event.getEvent(evt);
		if (e) {
		/**
		 * @default document.activeElement
		 */
			return  e.currentTarget || document.activeElement;
		} else {
			return null;
		}
	},

	/**
	 * 绂佹浜嬩欢鍐掓场浼犳挱
	 *
	 * @param {Event} evt 浜嬩欢锛岄潪蹇呰鍙傛暟
	 * @example QZFL.event.cancelBubble();
	 */
	cancelBubble : function(evt) {
		evt = QZFL.event.getEvent(evt);
		if (!evt) {
			return false
		}
		if (evt.stopPropagation) {
			evt.stopPropagation();
		} else {
			if (!evt.cancelBubble) {
				evt.cancelBubble = true;
			}
		}
	},

	/**
	 * 鍙栨秷娴忚鍣ㄧ殑榛樿浜嬩欢
	 *
	 * @param {Event} evt 浜嬩欢锛岄潪蹇呰鍙傛暟
	 * @example QZFL.event.preventDefault();
	 */
	preventDefault : function(evt) {
		evt = QZFL.event.getEvent(evt);
		if (!evt) {
			return false
		}
		if (evt.preventDefault) {
			evt.preventDefault();
		} else {
			evt.returnValue = false;
		}
	},

	/**
	 * 鑾峰彇浜嬩欢瑙﹀彂鏃剁殑榧犳爣浣嶇疆x
	 *
	 * @param {Object} evt 浜嬩欢瀵硅薄寮曠敤
	 * @example QZFL.event.mouseX();
	 */
	mouseX : function(evt) {
		evt = QZFL.event.getEvent(evt);
		return evt.pageX || (evt.clientX + (document.documentElement.scrollLeft || document.body.scrollLeft));
	},

	/**
	 * 鑾峰彇浜嬩欢瑙﹀彂鏃剁殑榧犳爣浣嶇疆y
	 *
	 * @param {Object} evt 浜嬩欢瀵硅薄寮曠敤
	 * @example QZFL.event.mouseX();
	 */
	mouseY : function(evt) {
		evt = QZFL.event.getEvent(evt);
		return evt.pageY || (evt.clientY + (document.documentElement.scrollTop || document.body.scrollTop));
	},

	/**
	 * 鑾峰彇浜嬩欢RelatedTarget
	 * @param {Object} evt 浜嬩欢瀵硅薄寮曠敤
	 * @example QZFL.event.getRelatedTarget();
	 */
	getRelatedTarget: function(ev) {
		ev = QZFL.event.getEvent(ev);
		var t = ev.relatedTarget;
		if (!t) {
			if (ev.type == "mouseout") {
				t = ev.toElement;
			} else if (ev.type == "mouseover") {
				t = ev.fromElement;
			} else {

			}
		}
		return t;
	},

	/**
	 * 鍏ㄥ眬椤甸潰鍔犺浇瀹屾垚鍚庣殑浜嬩欢鍥炶皟
	 * @param {function} fn 鍥炶皟鎺ュ彛
	 * @deprecated
	 */
	onDomReady:function(fn){
		var _s = QZFL.event.onDomReady;
		QZFL.event._bindReady();
		_s.pool.push(fn);
	},

	_bindReady:function(){
		var _s = QZFL.event.onDomReady;
		if(typeof _s.pool!='undefined'){//宸茬粡缁戝畾浜嗙洃鍚嚱鏁帮紝涓嶇敤鍐嶆缁戝畾
			return;
		}
		_s.pool = _s.pool || [];

		if(document.readyState === "complete"){
			return setTimeout(QZFL.event._readyFn, 1);
		}

		if(document.addEventListener){//Chrome Safari Firefox
			document.addEventListener("DOMContentLoaded",QZFL.event._domReady, false );
			window.addEventListener("load",QZFL.event._readyFn,false);
		}else if(document.attachEvent){//ie
			document.attachEvent("onreadystatechange",QZFL.event._domReady);
			window.attachEvent("onload",QZFL.event._readyFn);
			var toplevel = false;
			try{
				toplevel = window.frameElement == null;
			}catch(e){}
			if(document.documentElement.doScroll && toplevel){
				QZFL.event._ieScrollCheck();
			}
		}
	},

	_readyFn : function(){
		var _s = QZFL.event.onDomReady;
		_s.isReady = true;
		while(_s.pool.length) {
			var fn = _s.pool.shift();
			QZFL.lang.isFunction(fn) && fn();
		}
		_s.pool.length == 0 && (_s._fn = null);
	},

	_domReady : function(){
		if(document.addEventListener){
			document.removeEventListener("DOMContentLoaded",QZFL.event._domReady,false);
			QZFL.event._readyFn();
		}else if(document.attachEvent){
			if(document.readyState === "complete"){
				document.detachEvent( "onreadystatechange",QZFL.event._domReady);
				QZFL.event._readyFn();
			}
		}
	},

	// The DOM ready check for IE
	_ieScrollCheck : function() {
		if(QZFL.event.onDomReady.isReady){
			return;
		}
		try{
			document.documentElement.doScroll("left");
		}catch(e){
			setTimeout(QZFL.event._ieScrollCheck, 1 );
			return;
		}
		QZFL.event._readyFn();
	},
	/**
     * @description 浠ｇ悊浜嬩欢鍑芥暟锛岀涓€娆′娇鐢ㄤ负寮傛杩囩▼
     * @param {HTMLElement} delegateDom 浠ｇ悊鑺傜偣
     * @param {String} selector 鐩爣閫夋嫨鍣?     * @param {String} eventType 浜嬩欢绫诲瀷
     * @param {Function} fn 浜嬩欢鍑芥暟
     * @param {Array} [argsArray] 鍙傛暟鏁扮粍
     * @example
     * QZFL.event.delegate($('container'), '.namecard', 'mouseenter', function(){}); //浠ｇ悊浜嬩欢
     */
	delegate : function(delegateDom, selector, eventType, fn, argsArray){
	    var path = location.protocol + '//' + QZFL.config.resourceDomain + "/ac/qzfl/release/expand/delegate.js?max_age=864000";
	    
        QZFL.imports(path, function(){
            QZFL.event.delegate(delegateDom, selector, eventType, fn, argsArray);
        });
	},
	/**
     * @description 鍙栨秷浜嬩欢浠ｇ悊
     * @param {HTMLElement} delegateDom 鍙栨秷浜嬩欢浠ｇ悊鐨勮妭鐐?     * @param {String} [selector] 閫夋嫨鍣?     * @param {String} [eventType] 浜嬩欢绫诲瀷
     * @param {Function} [fn] 鍥炶皟鍑芥暟
     * @example
     * QZFL.event.undelegate($('container'));  //鍙栨秷鑺傜偣鐨勬墍鏈塪elegate浜嬩欢
     * QZFL.event.undelegate($('container'), 'click'); //鍙栨秷鑺傜偣鐨刢lick delegate浜嬩欢
     * QZFL.event.undelegate($('container'), '.namecard', 'click', fn); //鍙栨秷鎸囧畾鐨刣elegate浜嬩欢
     */
	undelegate : function(delegateDom, selector, eventType, fn){}
};

/**
 * 鏂规硶鍚?QZFL.event.addEvent
 *
 * @see QZFL.event.addEvent
 */
QZFL.event.on = QZFL.event.addEvent;

/**
 * 鏂规硶鍚?QZFL.object.bind
 *
 * @see QZFL.object.bind
 */
QZFL.event.bind = QZFL.object.bind;


/**
 * getEvent鏂规硶鐨勬渶娣遍€掑綊鏌ヨ灞傛
 * @ignore
 */
QZFL.event.getEvent.MAX_LEVEL = 10;




/////////////
//queue.js
/////////////


/**
 * @fileoverview QZFL 鍑芥暟闃熷垪绯荤粺锛屽彲浠ユ妸涓€绯诲垪鍑芥暟浣滀负闃熷垪骞朵笖鎸夐『搴忔墽琛屻€傚湪鎵ц杩囩▼涓嚱鏁板嚭鐜扮殑閿欒涓嶄細褰卞搷鍒颁笅涓€涓槦鍒楄繘绋? * @version 1.$Rev: 1921 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */

/**
 * 鍑芥暟闃熷垪寮曟搸
 *
 * @param {string} key 闃熷垪鍚嶇О
 * @param {array} queue 闃熷垪鍑芥暟鏁扮粍
 * @example QZFL.queue("test",[function(){alert(d)},function(){alert(2)}]);
 * QZFL.queue.run("test");
 *
 * @namespace QZFL 闃熷垪寮曟搸锛岀粰鍑芥暟鎻愪緵鎵归噺鐨勯槦鍒楁墽琛屾柟娉? * @return {Queue} 杩斿洖闃熷垪绯荤粺鏋勯€犲璞? */
QZFL.queue = (function(){
	var _o = QZFL.object;
	var _queue = {};

	var _Queue = function(key,queue){
		if (this instanceof arguments.callee) {
			this._qz_queuekey = key;
			return this;
		}

		if (_o.getType(queue = queue || []) == "array"){
			_queue[key] = queue;
		}

		return new _Queue(key);
	};

	var _extend = /**@lends QZFL.queue*/{
		/**
		 * 寰€涓€涓槦鍒楅噷鎻掑叆涓€涓柊鐨勫嚱鏁?		 *
		 * @param {string|function} key 闃熷垪鍚嶇О 褰撲綔涓烘瀯閫犲嚱鏁版椂鍒欏彧闇€瑕佺洿鎺ヤ紶
		 * @param {function} fn 鍙墽琛岀殑鍑芥暟
		 * @example QZFL.queue("test");
		 * QZFL.queue.push("test",function(){alert("ok")});
		 * // 鎴栬€?		 * QZFL.queue("test").push(function(){alert("ok")});
		 */
		push : function(key,fn){
			fn = this._qz_queuekey?key:fn;
			_queue[this._qz_queuekey || key].push(fn);
		},

		/**
		 * 浠庨槦鍒楅噷鍘婚櫎绗竴涓嚱鏁帮紝骞朵笖鎵ц涓€娆?		 *
		 * @param {string} key 闃熷垪鍚嶇О
		 * @example	QZFL.queue("test",[function(){alert("ok")}]);
		 * QZFL.queue.shift("test");
		 * // 鎴栬€?		 * QZFL.queue("test",[function(){alert("ok")}]).shift();
		 * @return 杩斿洖绗竴涓槦鍒楀嚱鏁版墽琛岀殑缁撴灉
		 */
		shift : function(key) {
			var _q = _queue[this._qz_queuekey || key];
			if (_q) {
				return QZFL.queue._exec(_q.shift());
			}
		},

		/**
		 * 杩斿洖闃熷垪闀垮害
		 * @param {string} key 闃熷垪鍚嶇О
		 *
		 * @example QZFL.queue("test",[function(){alert("ok")}]);
		 * QZFL.queue.getLen("test");
		 *      // 鎴栬€?		 * QZFL.queue("test",[function(){alert("ok")}]).getLen();
		 *
		 * @return 杩斿洖绗竴涓槦鍒楀嚱鏁版墽琛岀殑缁撴灉
		 */
		getLen: function(key){
			return _queue[this._qz_queuekey || key].length;
		},

		/**
		 * 鎵ц闃熷垪
		 *
		 * @param {string} key 闃熷垪鍚嶇О
		 * @example QZFL.queue("test",[function(){alert("ok")}]);
		 * QZFL.queue.run("test");
		 * // 鎴栬€?		 * QZFL.queue("test",[function(){alert("ok")}]).run();
		 */
		run : function(key){
			var _q = _queue[this._qz_queuekey || key];
			if (_q) {
				_o.each(_q,QZFL.queue._exec);
			}
		},

		/**
		 * 鍒嗘椂鎵ц闃熷垪
		 *
		 * @param {string} key 闃熷垪鍚嶇О
		 * @param {object} conf 鍙€夊弬鏁?榛樿鍊间负{'run': 100, 'wait': 50};姣忔杩愯100ms,鏆傚仠50ms鍐嶇户缁繍琛岄槦鍒?鐩磋嚦闃熷垪涓虹┖
		 * @example QZFL.queue("test",[function(){alert("1")},function(){alert("2")},function(){alert("3")}]);
		 * QZFL.queue.timedChunk("test", {'runTime': 1000, 'waitTime': 40, 'onRunEnd': function(){alert('allRuned');}, 'onWait': function(){alert('wait');}});
		 *
		 */
		timedChunk : function(key, conf){
			var _q = _queue[this._qz_queuekey || key], _conf;
			if (_q) {
				//鍚堝苟鐢ㄦ埛浼犲叆鐨勫弬鏁板拰榛樿鍙傛暟
				_conf = QZFL.lang.propertieCopy(conf, QZFL.queue._tcCof, null, true);
				setTimeout(function(){
					var _start = +new Date();
					do {
						QZFL.queue.shift(key);
					} while (QZFL.queue.getLen(key) > 0 && (+new Date() - _start < _conf.runTime));

					if (QZFL.queue.getLen(key) > 0){
						setTimeout(arguments.callee, _conf.waitTime);
						_conf.onWait();
					} else {
						_conf.onRunEnd();
					}
				}, 0);
			}
		},

		/**
		 * 鍒嗘椂鎵ц闃熷垪鐨勯粯璁ゅ弬鏁?		 *
		 */
		_tcCof : {
				'runTime': 50, //姣忔闃熷垪杩愯鏃堕棿
				'waitTime': 25, //鏆傚仠鏃堕棿
				'onRunEnd': QZFL.emptyFn,//闃熷垪鍏ㄩ儴杩愯瀹屾瘯瑙﹀彂鐨勪簨浠讹紙鍙Е鍙戜竴娆★級
				'onWait': QZFL.emptyFn//姣忔鏆傚仠鏃惰Е鍙戠殑浜嬩欢锛堣Е鍙戝娆★紝鏈夊彲鑳戒负闆舵锛?		},

		/**
		 *
		 */
		_exec : function(value,key,source){
			if (!value || _o.getType(value) != "function"){
				if (_o.getType(key) == "number") {
					source[key] = null;
				}
				return false;
			}

			try {
				return value();
			}catch(e){
				QZFL.console.print("QZFL Queue Got An Error: [" + e.name + "]  " + e.message,1)
			}
		}
	};

	_o.extend(_Queue.prototype,_extend);
	_o.extend(_Queue,_extend);

	return _Queue;
})();



/////////////
//string.js
/////////////


/**
 * @fileoverview QZFL String 缁勪欢
 * @version 1.$Rev: 1392 $
 * @author QzoneWebGroup, ($LastChangedBy: zishunchen $)
 */

/**
 * @namespace QZFL String 灏佽鎺ュ彛銆? * @type
 */
QZFL.string = {
	RegExps: {
		trim: /^\s+|\s+$/g,
		ltrim: /^\s+/,
		rtrim: /\s+$/,
		nl2br: /\n/g,
		s2nb: /[\x20]{2}/g,
		URIencode: /[\x09\x0A\x0D\x20\x21-\x29\x2B\x2C\x2F\x3A-\x3F\x5B-\x5E\x60\x7B-\x7E]/g,
		escHTML: {
			re_amp: /&/g,
			re_lt : /</g,
			re_gt : />/g,
			re_apos : /\x27/g,
			re_quot : /\x22/g
		},
		
		escString: {
			bsls: /\\/g,
			sls: /\//g,
			nl: /\n/g,
			rt: /\r/g,
			tab: /\t/g
		},
		
		restXHTML: {
			re_amp: /&amp;/g,
			re_lt: /&lt;/g,
			re_gt: /&gt;/g,
			re_apos: /&(?:apos|#0?39);/g,
			re_quot: /&quot;/g
		},
		
		write: /\{(\d{1,2})(?:\:([xodQqb]))?\}/g,
		isURL : /^(?:ht|f)tp(?:s)?\:\/\/(?:[\w\-\.]+)\.\w+/i,
		cut: /[\x00-\xFF]/,
		
		getRealLen: {
			r0: /[^\x00-\xFF]/g,
			r1: /[\x00-\xFF]/g
		},
		format: /\{([\d\w\.]+)\}/g
	},
	
	/**
	 * 閫氱敤鏇挎崲
	 *
	 * @ignore
	 * @param {string} s 闇€瑕佽繘琛屾浛鎹㈢殑瀛楃涓?	 * @param {String/RegExp} p 瑕佹浛鎹㈢殑妯″紡鐨?RegExp 瀵硅薄
	 * @param {string} r 涓€涓瓧绗︿覆鍊笺€傝瀹氫簡鏇挎崲鏂囨湰鎴栫敓鎴愭浛鎹㈡枃鏈殑鍑芥暟銆?	 * @example
	 * 			QZFL.string.commonReplace(str + "", QZFL.string.RegExps.trim, '');
	 * @returns {string} 澶勭悊缁撴灉
	 */
	commonReplace : function(s, p, r) {
		return s.replace(p, r);
	},
	
	/**
	 * 閫氱敤绯诲垪鏇挎崲
	 *
	 * @ignore
	 * @param {string} s 闇€瑕佽繘琛屾浛鎹㈢殑瀛楃涓?	 * @param {Object} l RegExp瀵硅薄hashMap
	 * @example
	 * 			QZFL.string.listReplace(str,regHashmap);
	 * @returns {string} 澶勭悊缁撴灉
	 */
	listReplace : function(s, l) {
		if (QZFL.lang.isHashMap(l)) {
			for (var i in l) {
				s = QZFL.string.commonReplace(s, l[i], i);
			}
			return s;
		} else {
			return s+'';
		}
	},
	
	/**
	 * 瀛楃涓插墠鍚庡幓绌烘牸
	 *
	 * @param {string} str 鐩爣瀛楃涓?	 * @example
	 * 			QZFL.string.trim(str);
	 * @returns {string} 澶勭悊缁撴灉
	 */
	trim: function(str){
		return QZFL.string.commonReplace(str + "", QZFL.string.RegExps.trim, '');
	},
	
	/**
	 * 瀛楃涓插墠鍘荤┖鏍?	 *
	 * @param {string} str 鐩爣瀛楃涓?	 * @example
	 * 			QZFL.string.ltrim(str);
	 * @returns {string} 澶勭悊缁撴灉
	 */
	ltrim: function(str){
		return QZFL.string.commonReplace(str + "", QZFL.string.RegExps.ltrim, '');
	},
	
	/**
	 * 瀛楃涓插悗鍘荤┖鏍?	 *
	 * @param {string} str 鐩爣瀛楃涓?	 * @example
	 * 			QZFL.string.rtrim(str);
	 * @returns {string} 澶勭悊缁撴灉
	 */
	rtrim: function(str){
		return QZFL.string.commonReplace(str + "", QZFL.string.RegExps.rtrim, '');
	},
	
	/**
	 * 鍒堕€爃tml涓崲琛岀
	 *
	 * @param {string} str 鐩爣瀛楃涓?	 * @example
	 * 			QZFL.string.nl2br(str);
	 * @returns {string} 缁撴灉
	 */
	nl2br: function(str){
		return QZFL.string.commonReplace(str + "", QZFL.string.RegExps.nl2br, '<br />');
	},
	
	/**
	 * 鍒堕€爃tml涓┖鏍肩锛岀埥鏇挎崲
	 *
	 * @param {string} str 鐩爣瀛楃涓?	 * @example
	 * 			QZFL.string.s2nb(str);
	 * @returns {string} 缁撴灉
	 */
	s2nb: function(str){
		return QZFL.string.commonReplace(str + "", QZFL.string.RegExps.s2nb, '&nbsp;&nbsp;');
	},
	
	/**
	 * 瀵归潪姹夊瓧鍋歎RIencode
	 *
	 * @param {string} str 鐩爣瀛楃涓?	 * @example
	 * 			QZFL.string.URIencode(str);
	 * @returns {string} 缁撴灉
	 */
	URIencode: function(str){
		var cc, ccc;
		return (str + "").replace(QZFL.string.RegExps.URIencode, function(a){
			if (a == "\x20") {
				return "+";
			} else if (a == "\x0D") {
				return "";
			}
			cc = a.charCodeAt(0);
			ccc = cc.toString(16);
			return "%" + ((cc < 16) ? ("0" + ccc) : ccc);
		});
	},
	
	/**
	 * htmlEscape
	 *
	 * @param {string} str 鐩爣涓?	 * @example
	 * 			QZFL.string.escHTML(str);
	 * @returns {string} 缁撴灉
	 */
	escHTML: function(str){
		var t = QZFL.string.RegExps.escHTML;
		return QZFL.string.listReplace((str + ""), {
		/*
		 * '&' must be
		 * escape first
		 */
			'&amp;' : t.re_amp,
			'&lt;' : t.re_lt,
			'&gt;' : t.re_gt,
			'&#039;' : t.re_apos,
			'&quot;' : t.re_quot
		});
	},
	
	/**
	 * CstringEscape
	 *
	 * @param {string} str 鐩爣涓?	 * @returns {string} 缁撴灉
	 */
	escString: function(str){
		var t = QZFL.string.RegExps.escString,
			h = QZFL.string.RegExps.escHTML;
		return QZFL.string.listReplace((str + ""), {
			/*
			 * '\' must be
			 * escape first
			 */
			'\\\\' : t.bsls,
			'\\n' : t.nl,
			'' : t.rt,
			'\\t' : t.tab,
			'\\/' : t.sls,
			'\\\'' : h.re_apos,
			'\\"' : h.re_quot
		});
	},
	
	/**
	 * htmlEscape杩樺師
	 *
	 * @param {string} str 鐩爣涓?	 * @returns {string} 缁撴灉
	 *
	restHTML: function(str){
		if (!QZFL.string.restHTML.__utilDiv) {
			/**
			 * 宸ュ叿DIV
			 *
			 * @ignore
			 *
			QZFL.string.restHTML.__utilDiv = document.createElement("div");
		}
		var t = QZFL.string.restHTML.__utilDiv;
		t.innerHTML = (str + "");
		if (typeof(t.innerText) != 'undefined') {
			return t.innerText;
		} else if (typeof(t.textContent) != 'undefined') {
			return t.textContent;
		} else if (typeof(t.text) != 'undefined') {
			return t.text;
		} else {
			return '';
		}
	},*/
	
	/**
	 * xhtmlEscape杩樺師
	 *
	 * @param {string} str 鐩爣涓?	 * @returns {string} 缁撴灉
	 */
	restXHTML: function(str){
		var t = QZFL.string.RegExps.restXHTML;
		return QZFL.string.listReplace((str + ""), {
			/*
			 * '&' must be
			 * escape last
			 */
			'<': t.re_lt,
			'>': t.re_gt,
			'\x27': t.re_apos,
			'\x22': t.re_quot,
			'&': t.re_amp
		});
	},
	
	/**
	 * 瀛楃涓叉牸寮忚緭鍑哄伐鍏?	 *
	 * @param {string} 杈撳嚭妯″紡
	 * @param {Arguments} Arguments... 鍙彉鍙傛暟锛岃〃绀烘ā寮忎腑鍗犱綅绗﹀瀹為檯瑕佹浛鎹㈢殑鍊?	 * @returns {string} 缁撴灉瀛楃涓?	 */
	write: function(strFormat, someArgs){
		if (arguments.length < 1 || !QZFL.lang.isString(strFormat)) {
			// rt.warn('No patern to write()');
			return '';
		}
		var rArr = QZFL.lang.arg2arr(arguments), result = rArr.shift(), tmp;
		
		return result.replace(QZFL.string.RegExps.write, function(a, b, c){
			b = parseInt(b, 10);
			if (b < 0 || (typeof rArr[b] == 'undefined')) {
				// rt.warn('write() wrong patern:{0:Q}', strFormat);
				return '(n/a)';
			} else {
				if (!c) {
					return rArr[b];
				} else {
					switch (c) {
						case 'x':
							return '0x' + rArr[b].toString(16);
						case 'o':
							return 'o' + rArr[b].toString(8);
						case 'd':
							return rArr[b].toString(10);
						case 'Q':
							return '\x22' + rArr[b].toString(16) + '\x22';
						case 'q':
							return '`' + rArr[b].toString(16) + '\x27';
						case 'b':
							return '<' + !!rArr[b] + '>';
					}
				}
			}
		});
	},
	
	/**
	 * 鏄惁鏄竴涓彲鎺ュ彈鐨刄RL涓?	 *
	 * @param {string} s 鐩爣涓?	 * @returns {Boolean} 缁撴灉
	 */
	isURL: function(s){
		return QZFL.string.RegExps.isURL.test(s);
	},
	
	
	/**
	 * 鍖呰鐨別scape鍑芥暟 <strong style="color:red;">Deprecated</strong>
	 *
	 * @param {string} s 婧愬瓧绗︿覆
	 * @returns {string} 缁撴灉涓?	 * @deprecated 鏈€鏃╂湡涓轰簡鏀寔寰堝樊娴忚鍣ㄧ殑鐢ㄦ硶锛岀幇鍦ㄤ笉闇€瑕佷簡
	 */
	escapeURI: function(s){
		if (window.encodeURIComponent) {
			return encodeURIComponent(s);
		}
		if (window.escape) {
			return escape(s);
		}
		return '';
	},
	
	/**
	 * 鐢ㄦ寚瀹氬瓧绗﹁ˉ瓒抽渶瑕佺殑鏁板瓧浣嶆暟
	 *
	 * @param {string} s 婧愬瓧绗︿覆
	 * @param {number} l 闀垮害
	 * @param {string} [ss="0"] 鎸囧畾瀛楃
	 * @param {boolean} [isBack=false] 琛ヨ冻鐨勬柟鍚? true 鍚庢柟; false 鍓嶆柟;
	 * @returns {string} 杩斿洖鐨勭粨鏋滀覆
	 */
	fillLength: function(source, l, ch, isRight){
		if ((source = String(source)).length < l) {
			var ar = new Array(l - source.length);
			ar[isRight ? 'unshift' : 'push'](source);
			source = ar.join(ch || '0');
		}
		return source;
	},
	/**
	 * 鐢ㄥ埗瀹氶暱搴﹀垏鍓茬粰瀹氬瓧绗︿覆
	 *
	 * @param {string} s 婧愬瓧绗︿覆
	 * @param {number} bl 鏈熸湜闀垮害(瀛楄妭闀垮害)
	 * @param {string} tails 澧炲姞鍦ㄦ渶鍚庣殑淇グ涓?姣斿"..."
	 * @returns {string} 缁撴灉涓?	 */
	cut: function(str, bitLen, tails){
		str = String(str);
		bitLen -= 0;
		tails = tails || '';
		if (isNaN(bitLen)) {
			return str;
		}
		var len = str.length,
			i = Math.min(Math.floor(bitLen / 2), len),
			cnt = QZFL.string.getRealLen(str.slice(0, i));

		for (; i < len && cnt < bitLen; i++) {
			cnt += 1 + (str.charCodeAt(i) > 255);
		}
		return str.slice(0, cnt > bitLen ? i - 1 : i) + (i < len ? tails : '');
	},
	
	/**
	 * 璁＄畻瀛楃涓茬殑鐪熷疄闀垮害
	 *
	 * @param {string} s 婧愬瓧绗︿覆
	 * @param {boolean} [isUTF8=false] 鏍囩ず鏄惁鏄痷tf-8璁＄畻
	 * @returns {number} 缁撴灉闀垮害
	 */
	getRealLen: function(s, isUTF8){
		if (typeof(s) != 'string') {
			return 0;
		}
		
		if (!isUTF8) {
			return s.replace(QZFL.string.RegExps.getRealLen.r0, "**").length;
		} else {
			var cc = s.replace(QZFL.string.RegExps.getRealLen.r1, "");
			return (s.length - cc.length) + (encodeURI(cc).length / 3);
		}
	},


	format: function(str){
		var args = Array.prototype.slice.call(arguments), v;
		str = String(args.shift());
		if (args.length == 1 && typeof(args[0]) == 'object') {
			args = args[0];
		}
		QZFL.string.RegExps.format.lastIndex = 0;
		return str.replace(QZFL.string.RegExps.format, function(m, n){
			v = QZFL.object.route(args, n);
			return v === undefined ? m : v;
		});
	}
};

/**
 * htmlEscape杩樺師
 * 
 * @deprecated 涓嶈鍐嶇敤浜嗭紝杩欓噷瀵规帴鐨勬柊鐨勫疄鐜颁笂
 * @param {string} str 鐩爣涓? * @returns {string} 缁撴灉
 */
QZFL.string.restHTML = QZFL.string.restXHTML;





/////////////
//util.js
/////////////


/**
 * @fileoverview QZFL 閫氱敤鎺ュ彛鏍稿績搴? * @version 1.$Rev: 1392 $
 * @author QzoneWebGroup, ($LastChangedBy: zishunchen $)
 * @lastUpdate $Date: 2009-08-05 16:26:13 +0800 (Wed, 05 Aug 2009) $
 */

/**
 * 灏忓伐鍏锋柟娉曞寘锛屼竴浜涘垎绫讳笉纭畾鐨勫叕鍏辨柟娉? * @namespace 灏忓伐鍏锋柟娉曞寘
 */
QZFL.util = {
	/**
	 * 浣跨敤涓€涓猽ri涓插埗浣滀竴涓被浼糽ocation鐨勫璞?	 *
	 * @param {string} s 鎵€闇€瀛楃涓?	 * @returns {object} QZFL.util.URI鐨勫疄渚?	 * @see QZFL.util.URI
	 */
	buildUri : function(s) {
		return new QZFL.util.URI(s);
	},

	/**
	 * 浣跨敤涓€涓猽ri涓插埗浣滀竴涓被浼糽ocation鐨勫璞?	 *
	 * @class URI寮曟搸锛屽彲浠ユ妸涓€涓猽ri瀛楃涓茶浆鎹㈡垚绫讳技location鐨勫璞?	 * @constructor
	 * @param {string} s 鎵€闇€瀛楃涓?	 */
	URI : function(s) {		
		if (!(QZFL.object.getType(s) == "string")) {
			return null;
		}
		if (s.indexOf("://") < 1) {
			s = location.protocol + "//" + location.hostname + (s.indexOf("/") == 0 ? "" : location.pathname.substr(0, location.pathname.lastIndexOf("/") + 1)) + s;
		}
		var depart = s.split("://");
		if (QZFL.object.getType(depart) == "array" && depart.length > 1 && (/^[a-zA-Z]+$/).test(depart[0])) {

			/**
			 * 鍗忚绫诲瀷
			 * @field
			 * @type string
			 */
			this.protocol = depart[0].toLowerCase();
			var h = depart[1].split("/");
			if (QZFL.object.getType(h) == "array") {

				/**
				 * 涓绘満鎻忚堪
				 * @field
				 * @type string
				 */
				this.host = h[0];

				/**
				 * 璧勬簮璺緞
				 * @field
				 * @type string
				 */
				this.pathname = "/" + h.slice(1).join("/").replace(/(\?|\#).+/i, ""); // 淇pathname鐨勮繑鍥為敊璇?
				/**
				 * 璇锋眰URL鍏ㄦ弿杩?				 * @field
				 * @type string
				 */
				this.href = s;
				var se = depart[1].lastIndexOf("?"), ha = depart[1].lastIndexOf("#");

				/**
				 * query string
				 * @field
				 * @type string
				 */
				this.search = (se >= 0) ? depart[1].substring(se) : "";

				/**
				 * page fragment anchor
				 * @field
				 * @type string
				 */
				this.hash = (ha >= 0) ? depart[1].substring(ha) : "";
				if (this.search.length > 0 && this.hash.length > 0) {
					if (ha < se) {
						this.search = "";
					} else {
						this.search = depart[1].substring(se, ha);
					}
				}
				return this;
			} else {
				return null;
			}
		} else {
			return null;
		}
	}
};



/////////////
//lang.js
/////////////


/**
 * @fileoverview 澧炲己鑴氭湰璇█澶勭悊鑳藉姏
 * @version 1.$Rev: 1597 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2009-11-30 21:51:19 +0800 (鏄熸湡涓€, 30 鍗佷竴鏈?2009) $
 */

 /**
 * 鐜鍙橀噺绯荤粺
 *
 * @namespace QZFL.lang
 */
QZFL.lang = {
	/**
	 * 鏄惁瀛楃涓?	 *
	 * @param {object} o 鐩爣
	 * @returns {boolean} 缁撴灉
	 * @example QZFL.lang.isString(obj);
	 */
	isString : function(o) {
		return QZFL.object.getType(o) == "string";
	},
	/**
	 * 鏄惁鏁扮粍瀵硅薄
	 *
	 * @param {object} o 鐩爣
	 * @returns {boolean} 缁撴灉
	 * @example QZFL.lang.isArray(obj);
	 */
	isArray : function(o) {
		return QZFL.object.getType(o) == "array";
	},
	/**
	 * 鏄惁鍑芥暟瀵硅薄
	 *
	 * @param {object} o 鐩爣
	 * @returns {boolean} 缁撴灉
	 * @example QZFL.lang.isArray(obj);
	 */
	isFunction: function(o) {
		return QZFL.object.getType(o) == "function";
	},
	/**
	 * 鏄惁鍝堝笇琛ㄧ粨鏋?	 *
	 * @param {object} o 鐩爣
	 * @returns {boolean} 缁撴灉
	 * @example QZFL.lang.isHashMap(obj);
	 */
	isHashMap : function(o) {
		return QZFL.object.getType(o) == "object";
	},

	/**
	 * 鏄惁Node鑺傜偣瀵硅薄
	 *
	 * @param {object} o 鐩爣
	 * @returns {boolean} 缁撴灉
	 * @example QZFL.lang.isNode(obj);
	 */
	isNode : function(o) {
		return o && (typeof(o.nodeName) != 'undefined' || typeof(o.nodeType) != 'undefined');
	},

	/**
	 * 鏄惁Element
	 *
	 * @param {object} o 鐩爣
	 * @returns {boolean} 缁撴灉
	 * @example QZFL.lang.isElement(obj);
	 */
	isElement : function(o) {
		 return o && o.nodeType == 1;
	}
};




/////////////
//util_ex.js
/////////////


/**
 * @fileoverview QZFL.util 灏忓伐鍏锋墿灞曞寘
 * @version 1.$Rev: 1921 $
 * @author QzoneWebGroup ($LastChangedBy: ryanzhao $) - $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */



(function() {
	var evalGlobalCnt = 0,
		runStyleGlobalCnt = 0;
	/**
	 * QZFL.util 宸ュ叿鍖呮墿灞?	 * @namespace QZFL.util 鍛藉悕绌洪棿
	 * @name QZFL.util
	 *
	 */
	QZFL.object.extend(QZFL.util, /** @lends QZFL.util */{
		/**
		 * 澶嶅埗鍒板壀璐存澘锛岀洰鍓嶅彧鏀寔IE 宸茬粡灏嗕笂涓増鏈琷olt澧炲姞鐨勫壀璐存澘鎺у埗鍘婚櫎
		 * @param {string} text 瑕佸鍒剁殑鏂囨湰
		 * @returns {boolean} 鍐欏叆鍓创鏉挎槸鍚︽垚鍔?		 * @deprecated 璁や负鍦ㄨ繖閲岄€昏緫杩囧,寤鸿璁捐缁熶竴鐨剋idget缁勪欢浜や粯鍚勪釜搴旂敤浣跨敤
		 * @example QZFL.util.copyToClip(text);
		 */
		copyToClip : function(text) {
			if (QZFL.userAgent.ie) {
				return clipboardData.setData("Text", text);
			} else {
				var o = QZFL.shareObject.getValidSO();
				return o ? o.setClipboard(text) : false;
			}
		},

		/**
		 * 鍦ㄩ〉闈笂鎵ц涓€娈礿s璇彞鏂囨湰
		 * 杩欎釜鏄洿鎺ュ湪鍏ㄥ眬閮ㄥ垎鎵ц锛屽湪js绯荤粺鐨勪换鎰忛儴鍒嗚皟鐢ㄤ篃鑳戒繚璇佸湪鍏ㄥ眬鎵ц涓€娈佃剼鏈?		 * @param {string} js 涓€娈礿s璇彞鏂囨湰
		 * @example QZFL.util.evalGlobal("var t = 1;");
		 */
		evalGlobal : function(js) {
			js = String(js);
			var obj = document.createElement('script'),
				head = document.documentElement || document.getElementsByTagName("head")[0];
			obj.type = 'text/javascript';
			obj.id = "__evalGlobal_" + evalGlobalCnt;
			try {
				obj.innerHTML = js;
			} catch (e) {
				obj.text = js;
			}
			head.insertBefore(obj,head.firstChild);
			evalGlobalCnt++;
			setTimeout(function(){QZFL.dom.removeElement(obj);}, 50);
		},

		/**
		 * 鍦ㄩ〉闈笂鎵ц涓€娈礳ss璇彞鏂囨湰
		 * @deprecated 涓撲緵safari浣跨敤
		 * @param {string} st 涓€娈祍tyle璇彞
		 * @example QZFL.util.runStyleGlobal("body { font-size: 12px; }");
		 */
		runStyleGlobal : function(st) {
			if (QZFL.userAgent.safari) {
				var obj = document.createElement('style');
				obj.type = 'text/css';
				obj.id = "__runStyle_" + runStyleGlobalCnt;
				try {
					obj.textContent = st;
				} catch (e) {
					alert(e.message);
				}
				var h = document.getElementsByTagName("head")[0];
				if (h) {
					h.appendChild(obj);
					runStyleGlobalCnt++;
				}
			} else {
			//	rt.warn("plz use runStyleGlobal() in Safari!");
			}
		},

		/**
		 * http鍙傛暟琛ㄥ璞″彉涓篐TTP鍗忚鏁版嵁涓诧紝濡傦細param1=123&amp;param2=456
		 * @param {object} o 鐢ㄦ潵琛ㄧず鍙傛暟鍒楄〃鐨刪ashTable
		 * @returns {string} 缁撴灉涓?		 * @example QZFL.util.genHttpParamString({"param1":123, "param2":456});
		 */
		genHttpParamString : function(o) {
			return QZFL.util.commonDictionaryJoin(o, null, null, null, window.encodeURIComponent);
		},

		/**
		 * 灏嗕竴涓猦ttp鍙傛暟搴忓垪瀛楃涓插彉涓鸿〃鏄犲皠瀵硅薄
		 * @param {string} s 婧愬瓧绗︿覆
		 * @returns {object} 缁撴灉
		 * @example QZFL.util.splitHttpParamString("param1=123&param2=456");
		 */
		splitHttpParamString : function(s) {
			return QZFL.util.commonDictionarySplit(s);
		},

		/**
		 * 灏嗕竴涓瓧鍏稿瀷搴忓垪瀛楃涓插彉涓烘槧灏勮〃瀵硅薄
		 * @param {string} [s=''] 婧愬瓧绗︿覆
		 * @param {string} [esp='&'] 椤瑰垎闅旂
		 * @param {string} [vq=''] 鍊煎皝濂?		 * @param {string} [eq='='] 绛夊彿瀛楃
		 * @returns {object} 缁撴灉瀵硅薄
		 * @example
QZFL.util.commonDictionarySplit(
	'form-data; name="file_upload"; file_name="c:\\data\\data.ini"; ',
	'; ',
	'"',
	'='
);
		 */
		commonDictionarySplit : function(s, esp, vq, eq) {
			var res = {},
				l,
				ks,
				vs,
				t;

			if(!s || typeof(s) != "string"){
				return res;
			}
			if (typeof(esp) != 'string') {
				esp = "&";
			}
			if (typeof(vq) != 'string') {
				vq = "";
			}
			if (typeof(eq) != 'string') {
				eq = "=";
			}

			l = s.split(esp); //a="1=2"tt"&b="2"s=t" -> a="1=2"tt"     b="2"s=t"

			if(l && l.length){
				for(var i = 0, len = l.length; i < len; ++i){
					ks = l[i].split(eq); //a="1=2"tt" -> a    "1    2"tt"
					if(ks.length > 1){
						t = ks.slice(1).join(eq); //"1=2"tt"
						vs = t.split(vq);
						res[ks[0]] = vs.slice(vq.length, vs.length - vq.length).join(vq);
					}else{
						ks[0] && (res[ks[0]] = true); //娌℃湁鍊肩殑鏃跺€欑洿鎺ュ氨鐢╰rue浣滀负鍊?					}
				}
			}						

			return res;
		},

		/**
		 * 灏嗕竴涓瓧鍏稿瀷鏄犲皠琛ㄥ璞″彉涓哄簭鍒楀瓧绗︿覆
		 * @param {object} [o={}] 婧愭槧灏勫璞?		 * @param {string} [esp='&'] 椤瑰垎闅旂
		 * @param {string} [vq=''] 鍊煎皝濂?		 * @param {string} [eq='='] 绛夊彿瀛楃
		 * @param {function} [valueHandler=QZFL.emptyFn] 澶勭悊鍊肩殑鏂规硶寮曠敤
		 * @returns {string} 缁撴灉涓?		 * @example
QZFL.util.commonDictionaryJoin(
	{
		'form-data' : true,
		'name' : 'file_upload',
		'file_name' : 'c:\\data\\data.ini'
	}
	'; ',
	'"',
	'='
); //form-data="true"; name="file_upload"; file_name="c:\\data\\data.ini"
		 */
		commonDictionaryJoin : function(o, esp, vq, eq, valueHandler) {
			var res = [],
				t;

			if(!o || typeof(o) != "object"){
				return '';
			}
			if(typeof(o) == "string"){
				return o;
			}
			if (typeof(esp) != 'string') {
				esp = "&";
			}
			if (typeof(vq) != 'string') {
				vq = "";
			}
			if (typeof(eq) != 'string') {
				eq = "=";
			}

			for(var k in o){
				res.push(k + eq + vq + (typeof valueHandler == 'function' ? valueHandler(o[k]) : o[k]) + vq);
			}

			return res.join(esp);
		}

	});

})();




/////////////
//lang_ex.js
/////////////


/**
 * @fileoverview 澧炲己鑴氭湰璇█澶勭悊鑳藉姏
 * @version 1.$Rev: 1921 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */

/**
 * 鏄惁鏄湁鏁堢殑xml鏁版嵁瀵硅薄
 *
 * @param {object} o xmldom瀵硅薄
 * @returns {boolean} 缁撴灉
 * @example QZFL.lang.isValidXMLdom(obj);
 */
QZFL.lang.isValidXMLdom = function(o) {
	return !!(o && o.xml && /^<\?xml/.test(o.xml)); //ryan
};

/**
 * 灏哸rguments瀵硅薄杞寲涓虹湡鏁扮粍
 *
 * @param {object} refArgs 瀵逛竴涓猘rguments瀵硅薄鐨勫紩鐢? * @param {number} [start=0] 璧峰鍋忕Щ閲? * @returns {object} 缁撴灉鏁扮粍
 * @example QZFL.lang.arg2arr(obj, 1); //浠庣浜屼釜鍙傛暟寮€濮嬭浆鍖? */
QZFL.lang.arg2arr = function(refArgs, start) {
	return Array.prototype.slice.call(refArgs, (start || 0));
};

/**
 * 浠indow涓烘牴,鑾峰彇鎸囧畾鍛藉悕鎻忚堪鐨勫€? *
 * @param {string} ns 鎻忚堪, 濡? QZFL.foo.bar
 * @param {boolean} [setup=false] 涓嶅瓨鍦ㄥ垯鍒涘缓
 * @example QZFL.lang.getObjByNameSpace("QZFL.foo.bar", true);
 * @returns {string|number|function|object|undefined} 鑾峰彇鍒板緱鍊? */
QZFL.lang.getObjByNameSpace = function(ns, setup) {
	if (typeof(ns) != 'string') {
		return ns;
	}
	var l = ns.split("."),
		r = window;

	try {
		for (var i = 0, len = l.length; i < len; ++i) {
			if(typeof(r[l[i]]) == 'undefined'){
				if(setup){
					r[l[i]] = {};
				}else{
					return;
				}
			}
			r = r[l[i]];
		}
		return r;
	} catch (ignore) {
		return;
	}
};

/**
 * JSON鏁版嵁娣卞害澶嶅埗
 *
 * @param {string|number|function|object|undefined} obj 闇€瑕佸鍒剁殑JSON鏁版嵁鏍归儴
 * @param {String} preventName 闇€瑕佽繃婊ょ殑瀛楁
 * @returns {string|number|function|object|undefined} 澶嶅埗鍑虹殑鏂癑SON鏁版嵁
 * @example QZFL.lang.objectClone(re, "msg");
 */
QZFL.lang.objectClone = function(obj, preventName) {
	if ((typeof obj) == 'object') {
		var res = (QZFL.lang.isArray(obj)) ? [] : {};
		for (var i in obj) {
			if (i != preventName)
				res[i] = QZFL.lang.objectClone(obj[i], preventName);
		}
		return res;
	} else if ((typeof obj) == 'function') {
		return Object;
	}
	return obj;
};


/**
 * JS Object convert to String
 * @param {string|number|function|object|undefined} obj
 * @returns {string} result
 * @example QZFL.lang.obj2str(obj);
 */
QZFL.lang.obj2str = function(obj) {
	var t, sw;

	if (typeof(obj) == 'object') {
		if(obj === null){ return 'null'; }

		if(window.JSON && window.JSON.stringify){
			return JSON.stringify(obj);
		}

		sw = QZFL.lang.isArray(obj);
		t = [];
		for (var i in obj) {
			t.push((sw ? "" : ("\"" + QZFL.string.escString(i) + "\":")) + obj2str(obj[i]));
		}
		t = t.join();
		return sw ? ("["+t+"]") : ("{"+t+"}");
	} else if (typeof(obj) == 'undefined') {
		return 'undefined';
	} else if (typeof(obj) == 'number' || typeof(obj) == 'function') {
		return obj.toString();
	}
	return !obj ? "\"\"" : ("\"" + QZFL.string.escString(obj) + "\"");
};

/**
 * 瀵硅薄鎴愬憳澶嶅埗(娴呰〃澶嶅埗)
 *
 * @param {object} s 澶嶅埗鐨勭洰鏍囧璞? * @param {object} b 澶嶅埗鐨勬簮瀵硅薄
 * @param {object} [propertiSet] 鎵€闇€瑕佺殑灞炴€у悕绉伴泦鍚? * @param {boolean} [notOverWrite=false] 涓嶅鍐? * @returns {object} 鐩爣瀵硅薄
 * @example QZFL.lang.propertieCopy(objt, objs, parray, false);
 */
QZFL.lang.propertieCopy = function(s, b, propertiSet, notOverWrite) {
	// 濡傛灉propertiSet == null 鎴栬€?!= Object锛屽垯浣跨敤b銆?	var l = (!propertiSet || typeof(propertiSet) != 'object') ? b : propertiSet;

	s = s || {};

	for (var p in l) {
		if (!notOverWrite || !(p in s)) {
			s[p] = l[p];
		}
	}

	return s;
};

/**
 * 椤哄簭鎵ц涓€绯诲垪鏂规硶锛屽緱鍒扮涓€涓垚鍔熸墽琛岀殑缁撴灉
 *
 * @param {function} arguments... 鍙彉鍙傛暟锛屼竴绯诲垪鍑芥暟鎵ц
 * @returns {string|number|function|object|undefined} 鎵ц缁撴灉
 * @example QZFL.lang.tryThese(functionOne, functionTwo, functionThree);
 */
QZFL.lang.tryThese = function(){
	for (var i = 0, len = arguments.length; i < len; ++i) {
		try {
			return arguments[i]();
		} catch (ign) {}
	}
	return;
};

/**
 * 灏嗕袱涓墽琛岃繃绋嬭繛鎺ヨ捣鏉ワ紝娉ㄦ剰锛岃繛鎺ュ悗涓嶅彲鍐嶅垎寮€锛屼笖鎵ц杩囩▼鐢˙oolean鍨嬫暟鎹爣璇嗘槸鍚︽垚鍔熸墽琛? *
 * @param {function} u 瑕佸厛鎵ц鐨勮繃绋? * @param {function} v 鍚庢墽琛岀殑杩囩▼
 * @returns {function} 杩炴帴鍚庣殑鎵ц杩囩▼
 * @example QZFL.lang.chain(functionOne, functionTwo);
 */
QZFL.lang.chain = function(u, v) {
	var calls = QZFL.lang.arg2arr(arguments);
	return function() {
		for (var i = 0, len = calls.length; i < len; ++i) {
			if (calls[i] && calls[i].apply(null, arguments) === false) {
				return false;
			}
		}
		return true;
	};
};

/**
 * 鍘婚櫎鏁扮粍涓噸澶嶇殑鍏冪礌
 *
 * @param {object} arr 婧愭暟缁? * @returns {object} 鍘婚櫎閲嶅鍏冪礌鍚庣殑鏁扮粍
 * @example QZFL.lang.uniqueArray(arr);
 */
QZFL.lang.uniqueArray = function(arr) {
	if(!QZFL.lang.isArray(arr)){ return arr; }
	var flag = {},index = 0;
	while (index < arr.length) {
		if (flag[arr[index]] == typeof(arr[index])) {
			arr.splice(index, 1);
			continue;
		}
		flag[arr[index].toString()] = typeof(arr[index]);
		++index;
	}

	return arr;
};





/////////////
//env.js
/////////////


/**
 * @fileoverview 灏侀棴闈欐€佺被ENV,绠＄悊鐜鍙橀噺
 * @version 1.$Rev: 1921 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */

/**
 * 鐜鍙橀噺绯荤粺
 *
 * @namespace QZFL.enviroment
 */
QZFL.enviroment = (function() {
	var _p = {},
		hookPool = {};

	/**
	 * 鑾峰彇鎸囧畾鐨勭幆澧冨彉閲?	 *
	 * @ignore
	 * @param {String} kname 鎸囧畾鐨勭幆澧冨彉閲忓悕绉?	 * @return {All} 瀛樺偍鍦ㄧ幆澧冨彉閲忎腑鐨勬暟鎹?	 * @example
	 * 			QZFL.enviroment.envGet(kname);
	 */
	function envGet(kname) {
		return _p[kname];
	}

	/**
	 * 鍒犻櫎鎸囧畾鐨勭幆澧冨彉閲?	 *
	 * @ignore
	 * @param {String} kname 鎸囧畾鐨勭幆澧冨彉閲忓悕绉?	 * @return {Boolean} 鏄惁鍒犻櫎鎴愬姛
	 * @example
	 * 			QZFL.enviroment.envDel(kname);
	 */
	function envDel(kname) {
		delete _p[kname];
		return true;
	}

	/**
	 * 浠ユ寚瀹氬悕绉拌缃幆澧冨彉閲?	 *
	 * @ignore
	 * @param {String} kname 鍚嶇О
	 * @param {All} value 鍊?	 * @return {Boolean} 鏄惁鎴愬姛
	 * @example
	 * 			QZFL.enviroment.envSet(kname,value);
	 */
	function envSet(kname, value) {
		if (typeof value == 'undefined') {
			if (typeof kname == 'undefined') {
				return false;
			} else if (!(_p[kname] === undefined)) {
				return false;
			}
		} else {
			_p[kname] = value;
			return true;
		}
	}

	return {
		/**
		 * 鑾峰彇鎸囧畾鐨勭幆澧冨彉閲?		 *
		 * @param {String} kname 鎸囧畾鐨勭幆澧冨彉閲忓悕绉?		 * @return {All} 瀛樺偍鍦ㄧ幆澧冨彉閲忎腑鐨勬暟鎹?		 * @memberOf QZFL.enviroment
		 */
		get : envGet,
		/**
		 * 浠ユ寚瀹氬悕绉拌缃幆澧冨彉閲?		 *
		 * @param {String} kname 鍚嶇О
		 * @param {All} value 鍊?		 * @return {Boolean} 鏄惁鎴愬姛
		 * @memberOf QZFL.enviroment
		 */
		set : envSet,
		/**
		 * 鍒犻櫎鎸囧畾鐨勭幆澧冨彉閲?		 *
		 * @param {String} kname 鎸囧畾鐨勭幆澧冨彉閲忓悕绉?		 * @return {Boolean} 鏄惁鍒犻櫎鎴愬姛
		 * @memberOf QZFL.enviroment
		 */
		del : envDel,
		/**
		 * 浜嬩欢閽╁瓙瀛樻斁鏍?		 *
		 * @type {Object}
		 * @memberOf QZFL.enviroment
		 */
		hookPool : hookPool
	};
})();

/**
 * 椤甸潰绾у埆浜嬩欢澶勭悊
 *
 * @namespace QZFL.pageEvents
 * @example
 * 		QZFL.pageEvents.pageBaseInit();
 *		QZFL.pageEvents.onloadRegister(init);
 */
QZFL.pageEvents = (function() {
	/**
	 * 灏唓ueryString瑙ｆ瀽鍒扮幆澧冨彉閲?	 *
	 * @ignore
	 */
	function _ihp() {
		var qs = location.search.substring(1),
			qh = location.hash.substring(1),
			s, h, n;

		ENV.set("_queryString", qs);
		ENV.set("_queryHash", qh);
		ENV.set("queryString", s = QZFL.util.splitHttpParamString(qs));
		ENV.set("queryHash", h = QZFL.util.splitHttpParamString(qh));

		//涓篞ZFL璁剧疆璋冭瘯绾у埆
		if(s && s.DEBUG){
			n = parseInt(s.DEBUG, 10);
			if (!isNaN(n)) {
				QZFL.config.debugLevel = n;
			}
		}

	}

	/**
	 * 椤甸潰鍚姩鍣?	 *
	 * @ignore
	 */
	function _bootStrap() {
		if(QZFL.event.onDomReady.isReady){
			setTimeout(_onloadHook,1);
		}else if (document.addEventListener) {
			document.addEventListener("DOMContentLoaded", _onloadHook, true);
		} else {
			var src = (window.location.protocol == 'https:') ? '//:' : 'javascript:void(0)';
			document.write('<script onreadystatechange="if(this.readyState==\'complete\'){this.parentNode.removeChild(this);QZFL.pageEvents._onloadHook();}" defer="defer" src="' + src + '"><\/script\>');
		}

		window.onload = QZFL.lang.chain(window.onload, function() {
			_onloadHook();
			_runHooks('onafterloadhooks');
		});
		/**
		 * 椤甸潰鐨刼nbeforeunload渚﹀惉
		 *
		 * @ignore
		 */
		window.onbeforeunload = function() {
			return _runHooks('onbeforeunloadhooks');
		};
		window.onunload = QZFL.lang.chain(window.onunload, function() {
			_runHooks('onunloadhooks');
		});
	}

	/**
	 * 鎵ц鎵€鏈塸age onload鏂规硶,骞朵笖缃爣蹇?	 */
	function _onloadHook() {
		_runHooks('onloadhooks');
		QZFL.enviroment.loaded = true;
		QZFL.event.onDomReady.isReady = true;
	}

	/**
	 * 鎵ц涓€涓寕閽╂柟娉?	 *
	 * @param {Function} handler 鎸囧畾鐨勬寕閽╂柟娉?	 */
	function _runHook(handler) {
		try {
			handler();
		} catch (ex) {
		}
	}

	/**
	 * 鎵ц鎵€鏈夋寚瀹氬悕绉扮殑鎸傞挬绋嬪簭
	 *
	 * @param {Object} hooks 鎸傞挬鍚嶇О
	 */
	function _runHooks(hooks) {
		var isbeforeunload = (hooks == 'onbeforeunloadhooks'),
			warn = null,
			hc = window.ENV.hookPool;

		do {
			var h = hc[hooks];
			if (!isbeforeunload) {
				hc[hooks] = null;
			}
			if (!h) {
				break;
			}
			for (var ii = 0; ii < h.length; ii++) {
				if (isbeforeunload) {
					warn = warn || h[ii]();
				} else {
					h[ii]();
				}
			}
			if (isbeforeunload) {
				break;
			}
		} while (hc[hooks]);

		if (isbeforeunload) {
			if (warn) {
				return warn;
			} else {
				QZFL.enviroment.loaded = false;
			}
		}
	}

	/**
	 * 澧炲姞浜嬩欢鎸傞挬
	 *
	 * @param {Object} hooks 鎸傞挬鍚嶇О
	 * @param {Function} handler 鐩爣鏂规硶
	 */
	function _addHook(hooks, handler) {
		var c = window.ENV.hookPool;
		(c[hooks] ? c[hooks] : (c[hooks] = [])).push(handler);
	}

	/**
	 * 鎻掑叆浜嬩欢鎸傞挬
	 *
	 * @param {Object} hooks 鎸傞挬鍚嶇О
	 * @param {Function} handler 鐩爣鏂规硶
	 * @param {Number} position 鐩爣浣嶇疆
	 */
	function _insertHook(hooks, handler, position) {
		var c = window.ENV.hookPool;
		if (typeof(position) == 'number' && position >= 0) {
			if(!c[hooks]){
				c[hooks] = [];
			}
			c[hooks].splice(position, 0, handler);
		} else {
			return false;
		}
	}

	/**
	 * 椤甸潰onload鏂规硶娉ㄥ唽
	 *
	 * @param {Function} handler 闇€瑕佹敞鍐岀殑椤甸潰onload鏂规硶寮曠敤
	 */
	function _lr(handler) {
		QZFL.enviroment.loaded ? _runHook(handler) : _addHook('onloadhooks', handler);
	}

	/**
	 * 椤甸潰onbeforeunload鏂规硶娉ㄥ唽
	 *
	 * @param {Function} handler 闇€瑕佹敞鍐岀殑椤甸潰onbeforeunload鏂规硶寮曠敤
	 */
	function _bulr(handler) {
		_addHook('onbeforeunloadhooks', handler);
	}

	/**
	 * 椤甸潰onunload鏂规硶娉ㄥ唽
	 *
	 * @param {Function} handler 闇€瑕佹敞鍐岀殑椤甸潰onunload鏂规硶寮曠敤
	 */
	function _ulr(handler) {
		_addHook('onunloadhooks', handler);
	}

	/**
	 * 椤甸潰鍒濆鍖栬繃绋?	 */
	function pinit() {
		_bootStrap();
		_ihp();

		/**
		 * 閿欒杈撳嚭
		 */
		var _dt;
		if (_dt = document.getElementById("__DEBUG_out")) {
			ENV.set("dout", _dt);
		}

		/**
		 * alert鏂规硶閲嶅畾涔?		 */
		var __dalert;
		if (!ENV.get("alertConverted")) {
			__dalert = alert;
			eval('var alert=function(msg){if(msg!=undefined){__dalert(msg);return msg;}}');// sds
			// 杩欓噷浠ュ悗鍙互鑰冭檻鏇村鏉傜殑閲嶅畾鍚?			ENV.set("alertConverted", true);
		}

		var t = ENV.get("queryHash");
		if(t && t.DEBUG){
			QZFL.config.debugLevel = 2;
		}
	}

	return {
		onloadRegister : _lr,
		onbeforeunloadRegister : _bulr,
		onunloadRegister : _ulr,
		initHttpParams : _ihp,
		bootstrapEventHandlers : _bootStrap,
		_onloadHook : _onloadHook,
		insertHooktoHooksQueue : _insertHook,
		pageBaseInit : pinit
	};
})();




/////////////
//string_ex.js
/////////////


/**
 * @fileOverview QZFL.string鎵╁睍鍖呯粍浠? * @version $Rev: 1921 $
 * @author QzoneWebGroup - ($LastChangedBy: ryanzhao $) - ($Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $)
 */


/**
 * 鍐呮兜鍚勭瀛楃涓插鐞嗙被宸ュ叿鎺ュ彛
 * @namespace QZFL.string鍚嶅瓧绌洪棿
 * @name QZFL.string
 */
QZFL.string = QZONE.string || {};


/**
 * 灏濊瘯瑙ｆ瀽涓€娈垫枃鏈负XML DOM鑺傜偣
 * @memberOf QZFL.string
 * @param {string} text 寰呰В鏋愮殑鏂囨湰
 * @returns {object} 杩斿洖缁撴灉,澶辫触鏄痭ull,鎴愬姛鏄痙ocumentElement
 */
QZFL.string.parseXML = function(text) {
	var doc;
	if (window.ActiveXObject) {
		doc = QZFL.lang.tryThese(function(){
			return new ActiveXObject('MSXML2.DOMDocument.6.0');
		}, function(){
			return new ActiveXObject('MSXML2.DOMDocument.5.0');
		}, function(){
			return new ActiveXObject('MSXML2.DOMDocument.4.0');
		}, function(){
			return new ActiveXObject('MSXML2.DOMDocument.3.0');
		}, function(){
			return new ActiveXObject('MSXML2.DOMDocument');
		}, function(){
			return new ActiveXObject('Microsoft.XMLDOM');
		});

		doc.async = "false";
		doc.loadXML(text);
		if (doc.parseError.reason) {
			// rt.error(doc.parseError.reason);
			return null;
		}
	} else {
		var parser = new DOMParser();
		doc = parser.parseFromString(text, "text/xml");
		if (doc.documentElement.nodeName == 'parsererror') {
			return null;
		}
	}
	return doc.documentElement;
};



/**
 * 鏍煎紡鍖栬緭鍑烘椂闂村伐鍏? * @param {number|object} date 姣鏁版弿杩扮殑缁濆鏄棿鍊?/ Date瀵硅薄寮曠敤
 * @param {string} [ptn=undefined] <strong style="color:green;">鑻ヤ笉缁欐鍙傛暟锛屽垯杩涘叆鑷姩鐩稿鏃堕棿妯″紡</strong><br /><br />
 鏍煎紡璇存槑涓?br />
 {y}涓や綅骞?br />
 {Y}鍥涗綅骞?br />
 {M}鏈?br />
 {d}鏃ユ湡<br />
 {h}灏忔椂<br />
 {m}鍒嗛挓鏁?br />
 {s}绉掓暟
 {_Y}鐩稿骞村樊鏁?br />
 {_M}鐩稿鏈堝樊鏁?br />
 {_d}鐩稿鏃ユ湡宸暟<br />
 {_h}鐩稿灏忔椂宸暟<br />
 {_m}鐩稿鍒嗛挓宸暟<br />
 {_s}鐩稿绉掑樊鏁? * @param {object} [baseTime=new Date()] 鐩稿鏃堕棿鍩哄噯瀵硅薄
 * @returns {string} 鏍煎紡杈撳嚭鐨勬枃鏈?
 * @example
var d0 = new Date(2011, 1, 26, 23, 4, 50),
	d1 = new Date(2011, 2, 5, 23, 4, 50);

function layout(){
	document.write(Array.prototype.join.apply(arguments, ['&lt;br />']));
}

layout(
	QZFL.string.timeFormatString(d1), //10澶╁墠   鍏跺疄鏄浉瀵逛簬褰撳墠鏃堕棿鐨勬櫤鑳藉亸绉绘彁绀?	QZFL.string.timeFormatString(d1, void(0), d0), //7澶╁墠
	QZFL.string.timeFormatString(d1, "{h}鏃秢m}鍒唟s}绉?), //23鏃?4鍒?0绉?
	QZFL.string.timeFormatString(d1, "{_s}鍒嗛挓鍓?, d0) //604800鍒嗛挓鍓?
);

 */
QZFL.string.timeFormatString = function(date, ptn, baseTime){
	try{
		date = date.getTime ? date : (new Date(date));
	}catch(ign){
		return '';
	}
	
	var me = QZFL.string.timeFormatString,
		map = me._map,
		unt = me._units,
		rel = false,
		t,
		delta,
		v;

	if(!ptn){
		baseTime = baseTime || new Date();

		delta = Math.abs(date - baseTime);
		for(var i = 0, len = unt.length; i < len; ++i){
			t = map[unt[i]];
			if(delta > t[1]){
				return Math.floor(delta / t[1]) + t[2];
			}
		}
		return "鍒氬垰";
	}else{
		return ptn.replace(me._re, function(a, b, c){
				(rel = b.charAt(0) == '_') && (b = b.charAt(1));
				if(!map[b]){
					return a;
				}
				if (!rel) {
					v = date[map[b][0]]();
					b == 'y' && (v %= 100);
					b == 'M' && v++;
					return v < 10 ? QZFL.string.fillLength(v, 2, c) : v.toString();
				} else {
					return Math.floor(Math.abs(date - baseTime) / map[b][1]);
				}
			});
	}
};
QZFL.string.timeFormatString._re = /\{([_yYMdhms]{1,2})(?:\:([\d\w\s]))?\}/g;
QZFL.string.timeFormatString._map = {	//sds 涓嶈鏇存敼
	y: ['getYear', 31104000000],
	Y: ['getFullYear', 31104000000, '\u5E74\u524D'],
	M: ['getMonth', 2592000000, '\u4E2A\u6708\u524D'],
	d: ['getDate', 86400000, '\u5929\u524D'],
	h: ['getHours', 3600000, '\u5C0F\u65F6\u524D'],
	m: ['getMinutes', 60000, '\u5206\u949F\u524D'],
	s: ['getSeconds', 1000, '\u79D2\u524D']
};
QZFL.string.timeFormatString._units = [	//sds 涓嶈鏇存敼
	'Y', 'M', 'd', 'h', 'm', 's'
];

/**
 * 瀛楃涓茶繛鍔犲櫒 deprecated
 *
 * @class 瀛楃涓茶繛鍔犲櫒
 * @constructor
 * @deprecated <strong style="color:red;">杩欎釜瀵硅薄鍜屾暟缁勮繛鎺ユ病浠€涔堝尯鍒紝寤鸿浠ュ悗涓嶈浣跨敤浜?/strong>
 */
QZFL.string.StringBuilder = function() {
	this._strList = QZFL.lang.arg2arr(arguments);
};

QZFL.string.StringBuilder.prototype = {
	/**
	 * 鍦ㄥ熬閮ㄥ鍔犱竴娈靛瓧绗︿覆
	 *
	 * @param {string} str 闇€瑕佸姞鍏ョ殑瀛楃涓?	 */
	append : function(str) {
		this._strList.push(String(str));
	},

	/**
	 * 鍦ㄦ渶鍓嶈拷鍔犱竴娈靛瓧绗︿覆
	 *
	 * @param {string} str 闇€瑕佸姞鍏ョ殑瀛楃涓?	 */
	insertFirst : function(str) {
		this._strList.unshift(String(str));
	},

	/**
	 * 澧炲姞涓€绯诲垪瀛楃涓?	 *
	 * @param {string[]} arr 闇€瑕佸姞鍏ョ殑瀛楃涓叉暟缁?	 */
	appendArray : function(arr) {
		this._strList = this._strList.concat(arr);
	},

	/**
	 * 绯诲垪鍖栨柟娉曞疄鐜?	 *
	 * @param {string} [spliter=""] 鐢ㄦ潵鍒嗗壊瀛楃缁勭殑绗﹀彿
	 * @returns {string} 瀛楃涓茶繛鍔犲櫒缁撴灉
	 */
	toString : function(spliter) {
		return this._strList.join(spliter || '');
	},

	/**
	 * 娓呯┖
	 */
	clear : function() {
		this._strList.splice(0, this._strList.length);
	}
};




/////////////
//sizzle_selector_engine_1.5.1.js
/////////////


/*!
 * Sizzle CSS Selector Engine
 *  Copyright 2011, The Dojo Foundation
 *  Released under the MIT, BSD, and GPL Licenses.
 *  More information: http://sizzlejs.com/
 */
(function(){

var chunker = /((?:\((?:\([^()]+\)|[^()]+)+\)|\[(?:\[[^\[\]]*\]|['"][^'"]*['"]|[^\[\]'"]+)+\]|\\.|[^ >+~,(\[\\]+)+|[>+~])(\s*,\s*)?((?:.|\r|\n)*)/g,
	done = 0,
	toString = Object.prototype.toString,
	hasDuplicate = false,
	baseHasDuplicate = true,
	rBackslash = /\\/g,
	rNonWord = /\W/,
	tmpVar,
	rSpeedUp = /^(\w+$)|^\.([\w\-]+$)|^#([\w\-]+$)|^(\w+)\.([\w\-]+$)/;

// Here we check if the JavaScript engine is using some sort of
// optimization where it does not always call our comparision
// function. If that is the case, discard the hasDuplicate value.
//   Thus far that includes Google Chrome.
[0, 0].sort(function() {
	baseHasDuplicate = false;
	return 0;
});

var Sizzle = function( selector, context, results, seed ) {
	results = results || [];
	context = context || document;

	var origContext = context;

	if ( context.nodeType !== 1 && context.nodeType !== 9 ) {
		return [];
	}
	
	if ( !selector || typeof selector !== "string" ) {
		return results;
	}

	var m, set, checkSet, extra, ret, cur, pop, i,
		prune = true,
		contextXML = Sizzle.isXML( context ),
		parts = [],
		soFar = selector,
		speedUpMatch;

	//sds speed up
	if ( !contextXML ) {
		//rSpeedUp.exec( "" ); //sds: do I need it ?
		speedUpMatch = rSpeedUp.exec( selector );

		if(speedUpMatch){//if(window.dbg){debugger}
			if ( context.nodeType === 1 || context.nodeType === 9 ) {
				// Speed-up: Sizzle("TAG")
				if ( speedUpMatch[1] ) {
					return makeArray( context.getElementsByTagName( selector ), results );
				
				// Speed-up: Sizzle(".CLASS") or Sizzle("TAG.CLASS")
				//} else if ( speedUpMatch[2] && Expr.find.CLASS ) {
				} else if ( speedUpMatch[2] || (speedUpMatch[4] && speedUpMatch[5]) ) {
					if(context.getElementsByClassName && speedUpMatch[2]){
						return makeArray( context.getElementsByClassName( speedUpMatch[2] ), results );
					}else{
						var suElems = context.getElementsByTagName(speedUpMatch[4] || '*'),
							suResBuff = [],
							suIt,
							suCN = ' ' + (speedUpMatch[2] || speedUpMatch[5]) + ' ';
						for(var sui = 0, sulen = suElems.length; sui < sulen; ++sui){
							suIt = suElems[sui];
							((' ' + suIt.className + ' ').indexOf(suCN) > -1) && suResBuff.push(suIt);
						}

						return makeArray( suResBuff, results );
					}
				}
			}

			if ( context.nodeType === 9 ) {
				// Speed-up: Sizzle("body")
				// The body element only exists once, optimize finding it
				if ( (selector === "body" || selector.toLowerCase() === "body") && context.body ) {
					return makeArray( [ context.body ], results );
					
				// Speed-up: Sizzle("#ID")
				} else if ( speedUpMatch[3] ) {
					return (tmpVar = context.getElementById( speedUpMatch[3] )) ? makeArray( [ tmpVar ], results ) : makeArray( [], results );

				//sds: no need, too much!
				/*	var suElem = context.getElementById( speedUpMatch[3] );

					// Check parentNode to catch when Blackberry 4.6 returns
					// nodes that are no longer in the document #6963
					if ( suElem && suElem.parentNode ) {
						// Handle the case where IE and Opera return items
						// by name instead of ID
						if ( suElem.id === speedUpMatch[3] ) {
							return makeArray( [ suElem ], results );
						}
						
					} else {
						return makeArray( [], results );
					} */
				}
			}
		}

	}//sds speed up end

	// Reset the position of the chunker regexp (start from head)
	do {
		chunker.exec( "" );
		m = chunker.exec( soFar );

		if ( m ) {
			soFar = m[3];
		
			parts.push( m[1] );
		
			if ( m[2] ) {
				extra = m[3];
				break;
			}
		}
	} while ( m );

	if ( parts.length > 1 && origPOS.exec( selector ) ) {

		if ( parts.length === 2 && Expr.relative[ parts[0] ] ) {
			set = posProcess( parts[0] + parts[1], context );

		} else {
			set = Expr.relative[ parts[0] ] ?
				[ context ] :
				Sizzle( parts.shift(), context );

			while ( parts.length ) {
				selector = parts.shift();

				if ( Expr.relative[ selector ] ) {
					selector += parts.shift();
				}
				
				set = posProcess( selector, set );
			}
		}

	} else {
		// Take a shortcut and set the context if the root selector is an ID
		// (but not if it'll be faster if the inner selector is an ID)
		if ( !seed && parts.length > 1 && context.nodeType === 9 && !contextXML &&
				Expr.match.ID.test(parts[0]) && !Expr.match.ID.test(parts[parts.length - 1]) ) {

			ret = Sizzle.find( parts.shift(), context, contextXML );
			context = ret.expr ?
				Sizzle.filter( ret.expr, ret.set )[0] :
				ret.set[0];
		}

		if ( context ) {
			ret = seed ?
				{ expr: parts.pop(), set: makeArray(seed) } :
				Sizzle.find( parts.pop(), parts.length === 1 && (parts[0] === "~" || parts[0] === "+") && context.parentNode ? context.parentNode : context, contextXML );

			set = ret.expr ?
				Sizzle.filter( ret.expr, ret.set ) :
				ret.set;

			if ( parts.length > 0 ) {
				checkSet = makeArray( set );

			} else {
				prune = false;
			}

			while ( parts.length ) {
				cur = parts.pop();
				pop = cur;

				if ( !Expr.relative[ cur ] ) {
					cur = "";
				} else {
					pop = parts.pop();
				}

				if ( pop == null ) {
					pop = context;
				}

				Expr.relative[ cur ]( checkSet, pop, contextXML );
			}

		} else {
			checkSet = parts = [];
		}
	}

	if ( !checkSet ) {
		checkSet = set;
	}

	if ( !checkSet ) {
		Sizzle.error( cur || selector );
	}

	if ( toString.call(checkSet) === "[object Array]" ) {
		if ( !prune ) {
			results.push.apply( results, checkSet );

		} else if ( context && context.nodeType === 1 ) {
			for ( i = 0; checkSet[i] != null; i++ ) {
				if ( checkSet[i] && (checkSet[i] === true || checkSet[i].nodeType === 1 && Sizzle.contains(context, checkSet[i])) ) {
					results.push( set[i] );
				}
			}

		} else {
			for ( i = 0; checkSet[i] != null; i++ ) {
				if ( checkSet[i] && checkSet[i].nodeType === 1 ) {
					results.push( set[i] );
				}
			}
		}

	} else {
		makeArray( checkSet, results );
	}

	if ( extra ) {
		Sizzle( extra, origContext, results, seed );
		Sizzle.uniqueSort( results );
	}

	return results;
};

Sizzle.uniqueSort = function( results ) {
	if ( sortOrder ) {
		hasDuplicate = baseHasDuplicate;
		results.sort( sortOrder );

		if ( hasDuplicate ) {
			for ( var i = 1; i < results.length; i++ ) {
				if ( results[i] === results[ i - 1 ] ) {
					results.splice( i--, 1 );
				}
			}
		}
	}

	return results;
};

Sizzle.matches = function( expr, set ) {
	return Sizzle( expr, null, null, set );
};

Sizzle.matchesSelector = function( node, expr ) {
	return Sizzle( expr, null, null, [node] ).length > 0;
};

Sizzle.find = function( expr, context, isXML ) {
	var set;

	if ( !expr ) {
		return [];
	}

	for ( var i = 0, l = Expr.order.length; i < l; i++ ) {
		var match,
			type = Expr.order[i];
		
		if ( (match = Expr.leftMatch[ type ].exec( expr )) ) {
			var left = match[1];
			match.splice( 1, 1 );

			if ( left.substr( left.length - 1 ) !== "\\" ) {
				match[1] = (match[1] || "").replace( rBackslash, "" );
				set = Expr.find[ type ]( match, context, isXML );

				if ( set != null ) {
					expr = expr.replace( Expr.match[ type ], "" );
					break;
				}
			}
		}
	}

	if ( !set ) {
		set = typeof context.getElementsByTagName !== "undefined" ?
			context.getElementsByTagName( "*" ) :
			[];
	}

	return { set: set, expr: expr };
};

Sizzle.filter = function( expr, set, inplace, not ) {
	var match, anyFound,
		old = expr,
		result = [],
		curLoop = set,
		isXMLFilter = set && set[0] && Sizzle.isXML( set[0] );

	while ( expr && set.length ) {
		for ( var type in Expr.filter ) {
			if ( (match = Expr.leftMatch[ type ].exec( expr )) != null && match[2] ) {
				var found, item,
					filter = Expr.filter[ type ],
					left = match[1];

				anyFound = false;

				match.splice(1,1);

				if ( left.substr( left.length - 1 ) === "\\" ) {
					continue;
				}

				if ( curLoop === result ) {
					result = [];
				}

				if ( Expr.preFilter[ type ] ) {
					match = Expr.preFilter[ type ]( match, curLoop, inplace, result, not, isXMLFilter );

					if ( !match ) {
						anyFound = found = true;

					} else if ( match === true ) {
						continue;
					}
				}

				if ( match ) {
					for ( var i = 0; (item = curLoop[i]) != null; i++ ) {
						if ( item ) {
							found = filter( item, match, i, curLoop );
							var pass = not ^ !!found;

							if ( inplace && found != null ) {
								if ( pass ) {
									anyFound = true;

								} else {
									curLoop[i] = false;
								}

							} else if ( pass ) {
								result.push( item );
								anyFound = true;
							}
						}
					}
				}

				if ( found !== undefined ) {
					if ( !inplace ) {
						curLoop = result;
					}

					expr = expr.replace( Expr.match[ type ], "" );

					if ( !anyFound ) {
						return [];
					}

					break;
				}
			}
		}

		// Improper expression
		if ( expr === old ) {
			if ( anyFound == null ) {
				Sizzle.error( expr );

			} else {
				break;
			}
		}

		old = expr;
	}

	return curLoop;
};

Sizzle.error = function( msg ) {
	throw "Syntax error, unrecognized expression: " + msg;
};

var Expr
	= Sizzle.selectors
	= {
	order: [ "ID", "NAME", "TAG" ],

	match: {
		ID: /#((?:[\w\u00c0-\uFFFF\-]|\\.)+)/,
		CLASS: /\.((?:[\w\u00c0-\uFFFF\-]|\\.)+)/,
		NAME: /\[name=['"]*((?:[\w\u00c0-\uFFFF\-]|\\.)+)['"]*\]/,
		ATTR: /\[\s*((?:[\w\u00c0-\uFFFF\-]|\\.)+)\s*(?:(\S?=)\s*(?:(['"])(.*?)\3|(#?(?:[\w\u00c0-\uFFFF\-]|\\.)*)|)|)\s*\]/,
		TAG: /^((?:[\w\u00c0-\uFFFF\*\-]|\\.)+)/,
		CHILD: /:(only|nth|last|first)-child(?:\(\s*(even|odd|(?:[+\-]?\d+|(?:[+\-]?\d*)?n\s*(?:[+\-]\s*\d+)?))\s*\))?/,
		POS: /:(nth|eq|gt|lt|first|last|even|odd)(?:\((\d*)\))?(?=[^\-]|$)/,
		PSEUDO: /:((?:[\w\u00c0-\uFFFF\-]|\\.)+)(?:\((['"]?)((?:\([^\)]+\)|[^\(\)]*)+)\2\))?/
	},

	leftMatch: {},

	attrMap: {
		"class": "className",
		"for": "htmlFor"
	},

	attrHandle: {
		href: function( elem ) {
			return elem.getAttribute( "href" );
		},
		type: function( elem ) {
			return elem.getAttribute( "type" );
		}
	},

	relative: {
		"+": function(checkSet, part){
			var isPartStr = typeof part === "string",
				isTag = isPartStr && !rNonWord.test( part ),
				isPartStrNotTag = isPartStr && !isTag;

			if ( isTag ) {
				part = part.toLowerCase();
			}

			for ( var i = 0, l = checkSet.length, elem; i < l; i++ ) {
				if ( (elem = checkSet[i]) ) {
					while ( (elem = elem.previousSibling) && elem.nodeType !== 1 ) {}

					checkSet[i] = isPartStrNotTag || elem && elem.nodeName.toLowerCase() === part ?
						elem || false :
						elem === part;
				}
			}

			if ( isPartStrNotTag ) {
				Sizzle.filter( part, checkSet, true );
			}
		},

		">": function( checkSet, part ) {
			var elem,
				isPartStr = typeof part === "string",
				i = 0,
				l = checkSet.length;

			if ( isPartStr && !rNonWord.test( part ) ) {
				part = part.toLowerCase();

				for ( ; i < l; i++ ) {
					elem = checkSet[i];

					if ( elem ) {
						var parent = elem.parentNode;
						checkSet[i] = parent.nodeName.toLowerCase() === part ? parent : false;
					}
				}

			} else {
				for ( ; i < l; i++ ) {
					elem = checkSet[i];

					if ( elem ) {
						checkSet[i] = isPartStr ?
							elem.parentNode :
							elem.parentNode === part;
					}
				}

				if ( isPartStr ) {
					Sizzle.filter( part, checkSet, true );
				}
			}
		},

		"": function(checkSet, part, isXML){
			var nodeCheck,
				doneName = done++,
				checkFn = dirCheck;

			if ( typeof part === "string" && !rNonWord.test( part ) ) {
				part = part.toLowerCase();
				nodeCheck = part;
				checkFn = dirNodeCheck;
			}

			checkFn( "parentNode", part, doneName, checkSet, nodeCheck, isXML );
		},

		"~": function( checkSet, part, isXML ) {
			var nodeCheck,
				doneName = done++,
				checkFn = dirCheck;

			if ( typeof part === "string" && !rNonWord.test( part ) ) {
				part = part.toLowerCase();
				nodeCheck = part;
				checkFn = dirNodeCheck;
			}

			checkFn( "previousSibling", part, doneName, checkSet, nodeCheck, isXML );
		}
	},

	find: {
		ID: function( match, context, isXML ) {
			if ( typeof context.getElementById !== "undefined" && !isXML ) {
				var m = context.getElementById(match[1]);
				// Check parentNode to catch when Blackberry 4.6 returns
				// nodes that are no longer in the document #6963
				return m && m.parentNode ? [m] : [];
			}
		},

		NAME: function( match, context ) {
			if ( typeof context.getElementsByName !== "undefined" ) {
				var ret = [],
					results = context.getElementsByName( match[1] );

				for ( var i = 0, l = results.length; i < l; i++ ) {
					if ( results[i].getAttribute("name") === match[1] ) {
						ret.push( results[i] );
					}
				}

				return ret.length === 0 ? null : ret;
			}
		},

		TAG: function( match, context ) {
			if ( typeof context.getElementsByTagName !== "undefined" ) {
				return context.getElementsByTagName( match[1] );
			}
		}
	},
	preFilter: {
		CLASS: function( match, curLoop, inplace, result, not, isXML ) {
			match = " " + match[1].replace( rBackslash, "" ) + " ";

			if ( isXML ) {
				return match;
			}

			for ( var i = 0, elem; (elem = curLoop[i]) != null; i++ ) {
				if ( elem ) {
					if ( not ^ (elem.className && (" " + elem.className + " ").replace(/[\t\n\r]/g, " ").indexOf(match) >= 0) ) {
						if ( !inplace ) {
							result.push( elem );
						}

					} else if ( inplace ) {
						curLoop[i] = false;
					}
				}
			}

			return false;
		},

		ID: function( match ) {
			return match[1].replace( rBackslash, "" );
		},

		TAG: function( match, curLoop ) {
			return match[1].replace( rBackslash, "" ).toLowerCase();
		},

		CHILD: function( match ) {
			if ( match[1] === "nth" ) {
				if ( !match[2] ) {
					Sizzle.error( match[0] );
				}

				match[2] = match[2].replace(/^\+|\s*/g, '');

				// parse equations like 'even', 'odd', '5', '2n', '3n+2', '4n-1', '-n+6'
				var test = /(-?)(\d*)(?:n([+\-]?\d*))?/.exec(
					match[2] === "even" && "2n" || match[2] === "odd" && "2n+1" ||
					!/\D/.test( match[2] ) && "0n+" + match[2] || match[2]);

				// calculate the numbers (first)n+(last) including if they are negative
				match[2] = (test[1] + (test[2] || 1)) - 0;
				match[3] = test[3] - 0;
			}
			else if ( match[2] ) {
				Sizzle.error( match[0] );
			}

			// TODO: Move to normal caching system
			match[0] = done++;

			return match;
		},

		ATTR: function( match, curLoop, inplace, result, not, isXML ) {
			var name = match[1] = match[1].replace( rBackslash, "" );
			
			if ( !isXML && Expr.attrMap[name] ) {
				match[1] = Expr.attrMap[name];
			}

			// Handle if an un-quoted value was used
			match[4] = ( match[4] || match[5] || "" ).replace( rBackslash, "" );

			if ( match[2] === "~=" ) {
				match[4] = " " + match[4] + " ";
			}

			return match;
		},

		PSEUDO: function( match, curLoop, inplace, result, not ) {
			if ( match[1] === "not" ) {
				// If we're dealing with a complex expression, or a simple one
				if ( ( chunker.exec(match[3]) || "" ).length > 1 || /^\w/.test(match[3]) ) {
					match[3] = Sizzle(match[3], null, null, curLoop);

				} else {
					var ret = Sizzle.filter(match[3], curLoop, inplace, true ^ not);

					if ( !inplace ) {
						result.push.apply( result, ret );
					}

					return false;
				}

			} else if ( Expr.match.POS.test( match[0] ) || Expr.match.CHILD.test( match[0] ) ) {
				return true;
			}
			
			return match;
		},

		POS: function( match ) {
			match.unshift( true );

			return match;
		}
	},
	
	filters: {
		enabled: function( elem ) {
			return elem.disabled === false && elem.type !== "hidden";
		},

		disabled: function( elem ) {
			return elem.disabled === true;
		},

		checked: function( elem ) {
			return elem.checked === true;
		},
		
		selected: function( elem ) {
			// Accessing this property makes selected-by-default
			// options in Safari work properly
			if ( elem.parentNode ) {
				elem.parentNode.selectedIndex;
			}
			
			return elem.selected === true;
		},

		parent: function( elem ) {
			return !!elem.firstChild;
		},

		empty: function( elem ) {
			return !elem.firstChild;
		},

		has: function( elem, i, match ) {
			return !!Sizzle( match[3], elem ).length;
		},

		header: function( elem ) {
			return (/h\d/i).test( elem.nodeName );
		},

		text: function( elem ) {
			// IE6 and 7 will map elem.type to 'text' for new HTML5 types (search, etc) 
			// use getAttribute instead to test this case
			return "text" === elem.getAttribute( 'type' );
		},
		radio: function( elem ) {
			return "radio" === elem.type;
		},

		checkbox: function( elem ) {
			return "checkbox" === elem.type;
		},

		file: function( elem ) {
			return "file" === elem.type;
		},
		password: function( elem ) {
			return "password" === elem.type;
		},

		submit: function( elem ) {
			return "submit" === elem.type;
		},

		image: function( elem ) {
			return "image" === elem.type;
		},

		reset: function( elem ) {
			return "reset" === elem.type;
		},

		button: function( elem ) {
			return "button" === elem.type || elem.nodeName.toLowerCase() === "button";
		},

		input: function( elem ) {
			return (/input|select|textarea|button/i).test( elem.nodeName );
		}
	},
	setFilters: {
		first: function( elem, i ) {
			return i === 0;
		},

		last: function( elem, i, match, array ) {
			return i === array.length - 1;
		},

		even: function( elem, i ) {
			return i % 2 === 0;
		},

		odd: function( elem, i ) {
			return i % 2 === 1;
		},

		lt: function( elem, i, match ) {
			return i < match[3] - 0;
		},

		gt: function( elem, i, match ) {
			return i > match[3] - 0;
		},

		nth: function( elem, i, match ) {
			return match[3] - 0 === i;
		},

		eq: function( elem, i, match ) {
			return match[3] - 0 === i;
		}
	},
	filter: {
		PSEUDO: function( elem, match, i, array ) {
			var name = match[1],
				filter = Expr.filters[ name ];

			if ( filter ) {
				return filter( elem, i, match, array );

			} else if ( name === "contains" ) {
				return (elem.textContent || elem.innerText || Sizzle.getText([ elem ]) || "").indexOf(match[3]) >= 0;

			} else if ( name === "not" ) {
				var not = match[3];

				for ( var j = 0, l = not.length; j < l; j++ ) {
					if ( not[j] === elem ) {
						return false;
					}
				}

				return true;

			} else {
				Sizzle.error( name );
			}
		},

		CHILD: function( elem, match ) {
			var type = match[1],
				node = elem;

			switch ( type ) {
				case "only":
				case "first":
					while ( (node = node.previousSibling) )	 {
						if ( node.nodeType === 1 ) { 
							return false; 
						}
					}

					if ( type === "first" ) { 
						return true; 
					}

					node = elem;

				case "last":
					while ( (node = node.nextSibling) )	 {
						if ( node.nodeType === 1 ) { 
							return false; 
						}
					}

					return true;

				case "nth":
					var first = match[2],
						last = match[3];

					if ( first === 1 && last === 0 ) {
						return true;
					}
					
					var doneName = match[0],
						parent = elem.parentNode;
	
					if ( parent && (parent.sizcache !== doneName || !elem.nodeIndex) ) {
						var count = 0;
						
						for ( node = parent.firstChild; node; node = node.nextSibling ) {
							if ( node.nodeType === 1 ) {
								node.nodeIndex = ++count;
							}
						} 

						parent.sizcache = doneName;
					}
					
					var diff = elem.nodeIndex - last;

					if ( first === 0 ) {
						return diff === 0;

					} else {
						return ( diff % first === 0 && diff / first >= 0 );
					}
			}
		},

		ID: function( elem, match ) {
			return elem.nodeType === 1 && elem.getAttribute("id") === match;
		},

		TAG: function( elem, match ) {
			return (match === "*" && elem.nodeType === 1) || elem.nodeName.toLowerCase() === match;
		},
		
		CLASS: function( elem, match ) {
			return (" " + (elem.className || elem.getAttribute("class")) + " ")
				.indexOf( match ) > -1;
		},

		ATTR: function( elem, match ) {
			var name = match[1],
				result = Expr.attrHandle[ name ] ?
					Expr.attrHandle[ name ]( elem ) :
					elem[ name ] != null ?
						elem[ name ] :
						elem.getAttribute( name ),
				value = result + "",
				type = match[2],
				check = match[4];

			return result == null ?
				type === "!=" :
				type === "=" ?
				value === check :
				type === "*=" ?
				value.indexOf(check) >= 0 :
				type === "~=" ?
				(" " + value + " ").indexOf(check) >= 0 :
				!check ?
				value && result !== false :
				type === "!=" ?
				value !== check :
				type === "^=" ?
				value.indexOf(check) === 0 :
				type === "$=" ?
				value.substr(value.length - check.length) === check :
				type === "|=" ?
				value === check || value.substr(0, check.length + 1) === check + "-" :
				false;
		},

		POS: function( elem, match, i, array ) {
			var name = match[2],
				filter = Expr.setFilters[ name ];

			if ( filter ) {
				return filter( elem, i, match, array );
			}
		}
	}
};

var origPOS = Expr.match.POS,
	fescape = function(all, num){
		return "\\" + (num - 0 + 1);
	};

for ( var type in Expr.match ) {
	Expr.match[ type ] = new RegExp( Expr.match[ type ].source + (/(?![^\[]*\])(?![^\(]*\))/.source) );
	Expr.leftMatch[ type ] = new RegExp( /(^(?:.|\r|\n)*?)/.source + Expr.match[ type ].source.replace(/\\(\d+)/g, fescape) );
}

var makeArray = function( array, results ) {
	array = Array.prototype.slice.call( array, 0 );

	if ( results ) {
		results.push.apply( results, array );
		return results;
	}
	
	return array;
};


//----------------------------------------just a check-----------------------------------

// Perform a simple check to determine if the browser is capable of
// converting a NodeList to an array using builtin methods.
// Also verifies that the returned array holds DOM nodes
// (which is not the case in the Blackberry browser)
try {
	Array.prototype.slice.call( document.documentElement.childNodes, 0 )[0].nodeType;

// Provide a fallback method if it does not work
} catch( e ) {
	makeArray = function( array, results ) {
		var i = 0,
			ret = results || [];

		if ( toString.call(array) === "[object Array]" ) {
			Array.prototype.push.apply( ret, array );

		} else {
			if ( typeof array.length === "number" ) {
				for ( var l = array.length; i < l; i++ ) {
					ret.push( array[i] );
				}

			} else {
				for ( ; array[i]; i++ ) {
					ret.push( array[i] );
				}
			}
		}

		return ret;
	};
}

//---------------------------------helper defined-------------------------------------

/**
 * @ignore
 */
var sortOrder,
	siblingCheck;

if ( document.documentElement.compareDocumentPosition ) {
	sortOrder = function( a, b ) {
		if ( a === b ) {
			hasDuplicate = true;
			return 0;
		}

		if ( !a.compareDocumentPosition || !b.compareDocumentPosition ) {
			return a.compareDocumentPosition ? -1 : 1;
		}

		return a.compareDocumentPosition(b) & 4 ? -1 : 1;
	};

} else {
	sortOrder = function( a, b ) {
		var al, bl,
			ap = [],
			bp = [],
			aup = a.parentNode,
			bup = b.parentNode,
			cur = aup;

		// The nodes are identical, we can exit early
		if ( a === b ) {
			hasDuplicate = true;
			return 0;

		// If the nodes are siblings (or identical) we can do a quick check
		} else if ( aup === bup ) {
			return siblingCheck( a, b );

		// If no parents were found then the nodes are disconnected
		} else if ( !aup ) {
			return -1;

		} else if ( !bup ) {
			return 1;
		}

		// Otherwise they're somewhere else in the tree so we need
		// to build up a full list of the parentNodes for comparison
		while ( cur ) {
			ap.unshift( cur );
			cur = cur.parentNode;
		}

		cur = bup;

		while ( cur ) {
			bp.unshift( cur );
			cur = cur.parentNode;
		}

		al = ap.length;
		bl = bp.length;

		// Start walking down the tree looking for a discrepancy
		for ( var i = 0; i < al && i < bl; i++ ) {
			if ( ap[i] !== bp[i] ) {
				return siblingCheck( ap[i], bp[i] );
			}
		}

		// We ended someplace up the tree so do a sibling check
		return i === al ?
			siblingCheck( a, bp[i], -1 ) :
			siblingCheck( ap[i], b, 1 );
	};

	siblingCheck = function( a, b, ret ) {
		if ( a === b ) {
			return ret;
		}

		var cur = a.nextSibling;

		while ( cur ) {
			if ( cur === b ) {
				return -1;
			}

			cur = cur.nextSibling;
		}

		return 1;
	};
}

// Utility function for retreiving the text value of an array of DOM nodes
Sizzle.getText = function( elems ) {
	var ret = "", elem;

	for ( var i = 0; elems[i]; i++ ) {
		elem = elems[i];

		// Get the text from text nodes and CDATA nodes
		if ( elem.nodeType === 3 || elem.nodeType === 4 ) {
			ret += elem.nodeValue;

		// Traverse everything else, except comment nodes
		} else if ( elem.nodeType !== 8 ) {
			ret += Sizzle.getText( elem.childNodes );
		}
	}

	return ret;
};


//--------------------------------------just a test-------------------------------------------

// Check to see if the browser returns elements by name when
// querying by getElementById (and provide a workaround)
(function(){
	// We're going to inject a fake input element with a specified name
	var form = document.createElement("div"),
		id = "script" + (new Date()).getTime(),
		root = document.documentElement;

	form.innerHTML = "<a name='" + id + "'/>";

	// Inject it into the root element, check its status, and remove it quickly
	root.insertBefore( form, root.firstChild );

	// The workaround has to do additional checks after a getElementById
	// Which slows things down for other browsers (hence the branching)
	if ( document.getElementById( id ) ) {
		Expr.find.ID = function( match, context, isXML ) {
			if ( typeof context.getElementById !== "undefined" && !isXML ) {
				var m = context.getElementById(match[1]);

				return m ?
					m.id === match[1] || typeof m.getAttributeNode !== "undefined" && m.getAttributeNode("id").nodeValue === match[1] ?
						[m] :
						undefined :
					[];
			}
		};

		Expr.filter.ID = function( elem, match ) {
			var node = typeof elem.getAttributeNode !== "undefined" && elem.getAttributeNode("id");

			return elem.nodeType === 1 && node && node.nodeValue === match;
		};
	}

	root.removeChild( form );

	// release memory in IE
	root = form = null;
})();


//------------------------------------------just a test------------------------------------

(function(){
	// Check to see if the browser returns only elements
	// when doing getElementsByTagName("*")

	// Create a fake element
	var div = document.createElement("div");
	div.appendChild( document.createComment("") );

	// Make sure no comments are found
	if ( div.getElementsByTagName("*").length > 0 ) {
		Expr.find.TAG = function( match, context ) {
			var results = context.getElementsByTagName( match[1] );

			// Filter out possible comments
			if ( match[1] === "*" ) {
				var tmp = [];

				for ( var i = 0; results[i]; i++ ) {
					if ( results[i].nodeType === 1 ) {
						tmp.push( results[i] );
					}
				}

				results = tmp;
			}

			return results;
		};
	}

	// Check to see if an attribute returns normalized href attributes
	div.innerHTML = "<a href='#'></a>";

	if ( div.firstChild && typeof div.firstChild.getAttribute !== "undefined" &&
			div.firstChild.getAttribute("href") !== "#" ) {

		Expr.attrHandle.href = function( elem ) {
			return elem.getAttribute( "href", 2 );
		};
	}

	// release memory in IE
	div = null;
})();


//----------------------------------------------querySelectorAll support--------------------------------

if ( document.querySelectorAll ) {
	(function(){
		var oldSizzle = Sizzle,
		//	div = document.createElement("div"),
			id = "__sizzle__";

		//div.innerHTML = "<p class='TEST'></p>";

		// Safari can't handle uppercase or unicode characters when
		// in quirks mode.

		//sds appended: 杩欎釜鏄庢槑safari鏈€鏂扮増鏈凡缁廸ix浜嗭紝涓嶈鍐嶆斁杩欎釜鎭跺績涓滀笢浜?		/*if ( div.querySelectorAll && div.querySelectorAll(".TEST").length === 0 ) {
			return;
		}*/
	
		Sizzle = function( query, context, extra, seed ) {
			context = context || document;

			// Only use querySelectorAll on non-XML documents
			// (ID selectors don't work in non-HTML documents)
			if ( !seed && !Sizzle.isXML(context) ) {
				// See if we find a selector to speed up
				var match = rSpeedUp.exec( query );
				
				if ( match && (context.nodeType === 1 || context.nodeType === 9) ) {
					// Speed-up: Sizzle("TAG")
					if ( match[1] ) {
						return makeArray( context.getElementsByTagName( query ), extra );
					
					// Speed-up: Sizzle(".CLASS")
					} else if ( match[2] && Expr.find.CLASS && context.getElementsByClassName ) {
						return makeArray( context.getElementsByClassName( match[2] ), extra );
					}
				}
				
				if ( context.nodeType === 9 ) {
					// Speed-up: Sizzle("body")
					// The body element only exists once, optimize finding it
					if ( query === "body" && context.body ) {
						return makeArray( [ context.body ], extra );
						
					// Speed-up: Sizzle("#ID")
					} else if ( match && match[3] ) {
						var elem = context.getElementById( match[3] );

						// Check parentNode to catch when Blackberry 4.6 returns
						// nodes that are no longer in the document #6963
						if ( elem && elem.parentNode ) {
							// Handle the case where IE and Opera return items
							// by name instead of ID
							if ( elem.id === match[3] ) {
								return makeArray( [ elem ], extra );
							}
							
						} else {
							return makeArray( [], extra );
						}
					}
					
					try {
						return makeArray( context.querySelectorAll(query), extra );
					} catch(qsaError) {}

				// qSA works strangely on Element-rooted queries
				// We can work around this by specifying an extra ID on the root
				// and working up from there (Thanks to Andrew Dupont for the technique)
				// IE 8 doesn't work on object elements
				} else if ( context.nodeType === 1 && context.nodeName.toLowerCase() !== "object" ) {
					var oldContext = context,
						old = context.getAttribute( "id" ),
						nid = old || id,
						hasParent = context.parentNode,
						relativeHierarchySelector = /^\s*[+~]/.test( query );

					if ( !old ) {
						context.setAttribute( "id", nid );
					} else {
						nid = nid.replace( /'/g, "\\$&" );
					}
					if ( relativeHierarchySelector && hasParent ) {
						context = context.parentNode;
					}

					try {
						if ( !relativeHierarchySelector || hasParent ) {
							return makeArray( context.querySelectorAll( "[id='" + nid + "'] " + query ), extra );
						}

					} catch(pseudoError) {
					} finally {
						if ( !old ) {
							oldContext.removeAttribute( "id" );
						}
					}
				}
			}
		
			return oldSizzle(query, context, extra, seed);
		};

		for ( var prop in oldSizzle ) {
			Sizzle[ prop ] = oldSizzle[ prop ];
		}

		// release memory in IE
		//div = null;
	})();
}

(function(){
	var html = document.documentElement,
		matches = html.matchesSelector || html.mozMatchesSelector || html.webkitMatchesSelector || html.msMatchesSelector,
		pseudoWorks = false;

	try {
		// This should fail with an exception
		// Gecko does not error, returns false instead
		matches.call( document.documentElement, "[test!='']:sizzle" );
	
	} catch( pseudoError ) {
		pseudoWorks = true;
	}

	if ( matches ) {
		Sizzle.matchesSelector = function( node, expr ) {
			// Make sure that attribute selectors are quoted
			expr = expr.replace(/\=\s*([^'"\]]*)\s*\]/g, "='$1']");

			if ( !Sizzle.isXML( node ) ) {
				try { 
					if ( pseudoWorks || !Expr.match.PSEUDO.test( expr ) && !/!=/.test( expr ) ) {
						return matches.call( node, expr );
					}
				} catch(e) {}
			}

			return Sizzle(expr, null, null, [node]).length > 0;
		};
	}
})();


//sds: remove this closure, too much!
//(function(){
//	var div = document.createElement("div");

//	div.innerHTML = "<div class='test e'></div><div class='test'></div>";

	// Opera can't find a second classname (in 9.6)
	// Also, make sure that getElementsByClassName actually exists
	//sds: removed, too much
	/*if ( !div.getElementsByClassName || div.getElementsByClassName("e").length === 0 ) {
		return;
	}*/

	// Safari caches class attributes, doesn't catch changes (in 3.2)
	//sds: removed, too much
	/*div.lastChild.className = "e";

	if ( div.getElementsByClassName("e").length === 1 ) {
		return;
	}*/
	
Expr.order.splice(1, 0, "CLASS");
Expr.find.CLASS = function( match, context, isXML ) {
	if ( typeof context.getElementsByClassName !== "undefined" && !isXML ) {
		return context.getElementsByClassName(match[1]);
	}
};

	// release memory in IE
//	div = null;
//})();

function dirNodeCheck( dir, cur, doneName, checkSet, nodeCheck, isXML ) {
	for ( var i = 0, l = checkSet.length; i < l; i++ ) {
		var elem = checkSet[i];

		if ( elem ) {
			var match = false;

			elem = elem[dir];

			while ( elem ) {
				if ( elem.sizcache === doneName ) {
					match = checkSet[elem.sizset];
					break;
				}

				if ( elem.nodeType === 1 && !isXML ){
					elem.sizcache = doneName;
					elem.sizset = i;
				}

				if ( elem.nodeName.toLowerCase() === cur ) {
					match = elem;
					break;
				}

				elem = elem[dir];
			}

			checkSet[i] = match;
		}
	}
}

function dirCheck( dir, cur, doneName, checkSet, nodeCheck, isXML ) {
	for ( var i = 0, l = checkSet.length; i < l; i++ ) {
		var elem = checkSet[i];

		if ( elem ) {
			var match = false;
			
			elem = elem[dir];

			while ( elem ) {
				if ( elem.sizcache === doneName ) {
					match = checkSet[elem.sizset];
					break;
				}

				if ( elem.nodeType === 1 ) {
					if ( !isXML ) {
						elem.sizcache = doneName;
						elem.sizset = i;
					}

					if ( typeof cur !== "string" ) {
						if ( elem === cur ) {
							match = true;
							break;
						}

					} else if ( Sizzle.filter( cur, [elem] ).length > 0 ) {
						match = elem;
						break;
					}
				}

				elem = elem[dir];
			}

			checkSet[i] = match;
		}
	}
}


if ( document.documentElement.compareDocumentPosition ) {
	Sizzle.contains = function( a, b ) {
		return !!(a.compareDocumentPosition(b) & 16);
	};

} else if ( document.documentElement.contains ) {
	Sizzle.contains = function( a, b ) { //sds hacked
		if(a !== b && a.contains && b.contains){
			return a.contains(b);
		}else if(!b || b.nodeType == 9){
			return false;
		}else if(b === a){
			return true;
		}else{
			return Sizzle.contains(a, b.parentNode);
		}
	};

} else {
	Sizzle.contains = function() {
		return false;
	};
}

Sizzle.isXML = function( elem ) {
	// documentElement is verified for cases where it doesn't yet exist
	// (such as loading iframes in IE - #4833) 
	var documentElement = (elem ? elem.ownerDocument || elem : 0).documentElement;

	return documentElement ? documentElement.nodeName !== "HTML" : false;
};

var posProcess = function( selector, context ) {
	var match,
		tmpSet = [],
		later = "",
		root = context.nodeType ? [context] : context;

	// Position selectors must be done after the filter
	// And so must :not(positional) so we move all PSEUDOs to the end
	while ( (match = Expr.match.PSEUDO.exec( selector )) ) {
		later += match[0];
		selector = selector.replace( Expr.match.PSEUDO, "" );
	}

	selector = Expr.relative[selector] ? selector + "*" : selector;

	for ( var i = 0, l = root.length; i < l; i++ ) {
		Sizzle( selector, root[i], tmpSet );
	}

	return Sizzle.filter( later, tmpSet );
};

// EXPOSE
QZFL.selector = window.Sizzle = Sizzle;
QZFL.object.makeArray = QZFL.dom.collection2Array = makeArray;
QZFL.dom.uniqueSort = Sizzle.uniqueSort;
QZFL.dom.contains = Sizzle.contains;


})();




/////////////
//element.js
/////////////


/**
 * @fileoverview QZFL Element瀵硅薄
 * @version 1.$Rev: 1921 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */

;
(function() {
	/**
	 * QZFL Element 鎺у埗鍣?閫氬父涓嶈璇疯嚜浼犲叆闈瀞tring鍙傛暟
	 *
	 * @param {string|elements} selector selector鏌ヨ璇彞,鎴栧垯涓€缁別lements瀵硅薄
	 * @param {element} context element鏌ヨ浣嶇疆
	 * @class QZFL Element 鎺у埗鍣?閫氬父涓嶈璇疯嚜浼犲叆闈瀞tring鍙傛暟
	 * @constructor
	 */
	var _handler = QZFL.ElementHandler = function(selector, context){
		/**
		 * 鏌ヨ鍒扮殑瀵硅薄鏁扮粍
		 *
		 * @type array
		 */
		this.elements = null;
		
		/**
		 * 鐢ㄦ潵鍋氫竴涓爣绀哄尯鍒?		 *
		 * @ignore
		 */
		this._isElementHandler = true;
		
		this._init(selector, context);
		
	};
	/** @lends QZFL.ElementHandler.prototype */
	_handler.prototype = {
		/**
		 * 鍒濆鍖?elementHandler瀵硅薄
		 * @private
		 * @param {Object} selector
		 * @param {Object} context
		 */
		_init: function(selector, context){
			if (QZFL.lang.isString(selector)) {
				this.elements = QZFL.selector(selector, context);
			} else if (selector instanceof QZFL.ElementHandler) {
				this.elements = selector.elements.slice();
			} else if (QZFL.lang.isArray(selector)) {
				this.elements = selector;
			} else if (selector && ((selector.nodeType && selector.nodeType !== 3 && selector.nodeType !== 8) || selector.setTimeout)) {
				this.elements = [selector];
			} else {
				this.elements = [];
			}
		},
		/**
		 * 鏌ユ壘 elements 瀵硅薄
		 *
		 * @param {string} selector selector鏌ヨ璇硶
		 *            @example
		 *            $e("div").findElements("li");
		 * @return {Array} elements 鏁扮粍
		 */
		findElements: function(selector){
			var _pushstack = [],_s;
			this.each(function(el){
				_s = QZFL.selector(selector, el);
				if (_s.length > 0) {
					_pushstack = _pushstack.concat(_s);
				}
			});
			return _pushstack;
		},

		/**
		 * 鏌ユ壘 elements ,骞朵笖鍒涘缓QZFL Elements 瀵硅薄.
		 *
		 * @param {string} selector selector鏌ヨ璇硶
		 *            @example
		 *            $e("div").find("li");
		 * @return {QZFL.ElementHandler}
		 */
		find: function(selector){
			return _el.get(this.findElements(selector));
		},
		filter: function(expr, elems, not){
			if (not) {
				expr = ":not(" + expr + ")";
			}
			return _el.get(QZFL.selector.matches(expr, elems||this.elements));
		},
		/**
		 * 寰幆鎵цelements瀵硅薄
		 *
		 * @param {function} fn 鎵归噺鎵ц鐨勬搷浣?		 *            @example
		 *            $e("div").each(function(n){alert("hello!" + n)});
		 */
		each: function(fn){
			QZFL.object.each(this.elements, fn);
			return this;
		},

		/**
		 * 鍜屽叾浠?Element Handler 鎴?elements Array 鍚堝苟
		 *
		 * @param {QZFL.ElementHandler|Array} elements Element Handler瀵硅薄鎴栧垯
		 *            Element 鏁扮粍闆嗗悎
		 *            @example
		 *            $e("div").concat($e("p"))
		 * @return {QZFL.ElementHandler}
		 */
		concat: function(elements){
			return _el.get(this.elements.concat(!!elements._isElementHandler ? elements.elements : elements));
		},

		/**
		 * 閫氳繃 index 鑾峰彇鍏朵腑涓€涓?Element Handler
		 *
		 * @param {number} index 绱㈠紩
		 * @return {QZFL.ElementHandler}
		 */
		get: function(index){
			return _el.get(this.elements[index]);
		},
		/**
		 * 鍙杋ndex鍏冪礌
		 */
		eq: function(index){
			return this.elements[index || 0];
		},
		/**
		 * 鍚剰鍚孉rray.prorotype.slice
		 * @param {number} index 绱㈠紩
		 * @return {QZFL.ElementHandler}
		 */
		slice: function(){
			return _el.get(Array.prototype.slice.apply(this.elements, arguments));
		}
	};
	/**
	 * QZFL Element瀵硅薄.
	 *
	 * @namespace QZFL element 瀵硅薄鐨勫墠绔帶鍒跺櫒
	 * @requires QZFL.selector
	 * @type Object
	 */
	var _el = QZFL.element = {
		/**
		 * 鑾峰彇 element 瀵硅薄
		 *
		 * @param {string} selector selector鏌ヨ璇彞
		 *            @example
		 *            QZFL.element.get("div")
		 * @param {element} context element鏌ヨ浣嶇疆
		 * @see QZFL.ElementHandler
		 * @return QZFL.ElementHandler
		 */
		get : function(selector, context) {
			return new _handler(selector, context);
		},

		/**
		 * 鎵╁睍 QZFL elements Handler 瀵硅薄鎺ュ彛
		 *
		 * @param {object} object 鎵╁睍鎺ュ彛
		 *            @example
		 *            QZFL.element.extend({show:function(){}})
		 */
		extend : function(object) {
			QZFL.object.extend(_handler, object);
		},

		/**
		 * 鎵╁睍 QZFL elements Handler 鏋勯€犲嚱鏁版帴鍙?		 *
		 * @param {object} object 鎵╁睍鎺ュ彛
		 *            @example
		 *            QZFL.element.extendFn({show:function(){}})
		 */
		extendFn : function(object) {
			QZFL.object.extend(_handler.prototype, object);
		},

		/**
		 * 杩斿洖 QZFL Elements 瀵硅薄鐨勭増鏈?		 *
		 * @return {string}
		 */
		getVersion : function() {
			return _handler.version;
		}
	}
})();


// 鎵╁睍 QZFL Element 鎺ュ彛
QZFL.element.extend(/** @lends QZFL.ElementHandler */
{
	/**
	 * QZFL Element 鐗堟湰
	 *
	 * @type String

	 */
	version : "1.0"
});

// Extend Events
QZFL.element.extendFn(
/** @lends QZFL.ElementHandler.prototype */
{
	/**
	 * 缁戝畾浜嬩欢
	 *
	 * @param {string} evtType 浜嬩欢绫诲瀷
	 * @param {function} fn 瑙﹀彂鍑芥暟
	 *            @example
	 *            $e("div.head").bind("click",function(){});
	 * @param {object - Array} argArr 浼犲叆浜嬩欢渚﹀惉鍣ㄧ殑鍙傛暟鍒楄〃
	 */
	bind : function(evtType, fn, argArr) {
		if(typeof(fn)!='function'){
			return false;
		}
		return this.each(function(el) {
			QZFL.event.addEvent(el, evtType, fn, argArr);
		});
	},

	/**
	 * 鍙栨秷浜嬩欢缁戝畾
	 *
	 * @param {string} evtType 浜嬩欢绫诲瀷
	 * @param {function} fn 瑙﹀彂鍑芥暟
	 *            @example
	 *            $e("div.head").unBind("click",function(){});
	 */
	unBind : function(evtType, fn) {
		return this.each(function(el) {
			QZFL.event[fn ? 'removeEvent' : 'purgeEvent'](el, evtType, fn);
		});
	},
	/**
	 * @param {} fn
	 */
	onHover : function(fnOver,fnOut) {
		this.onMouseOver( fnOver);
		return this.onMouseOut( fnOut);
	},
	onMouseEnter:function(fn){
		return this.bind('mouseover',function(evt){
			var rel = QZFL.event.getRelatedTarget(evt); // fromElement
			if(QZFL.lang.isFunction(fn) && !QZFL.dom.isAncestor(this,rel)){
				fn.call(this,evt);
			}
		});
	},
	onMouseLeave:function(fn){
		return this.bind('mouseout',function(evt){
			var rel = QZFL.event.getRelatedTarget(evt); // toElement
			if(QZFL.lang.isFunction(fn) && !QZFL.dom.isAncestor(this,rel)){
				fn.call(this,evt);
			}
		});
	},
	delegate:function(selector, eventType, fn, argsArray){
	    if(typeof(fn) != 'function'){
            return false;
        }
	    return this.each(function(el) {
            QZFL.event.delegate(el, selector, eventType, fn, argsArray);
        });
	},
	undelegate:function(selector, eventType, fn){
	    return this.each(function(el){
            QZFL.event.undelegate(el, selector, eventType, fn);
        });
	}
});
QZFL.object.each(['onClick', 'onMouseDown', 'onMouseUp', 'onMouseOver', 'onMouseMove', 'onMouseOut', 'onFocus', 'onBlur', 'onKeyDown', 'onKeyPress', 'onKeyUp'], function(name, index){
	QZFL.ElementHandler.prototype[name] = function(fn){
		return this.bind(name.slice(2).toLowerCase(), fn);
	};
});
// Extend Dom
QZFL.element.extendFn(
/** @lends QZFL.ElementHandler.prototype */
{

	/**
	 * 璁剧疆 dom 鐨刪tml浠ｇ爜
	 *
	 * @param {string} value
	 */
	setHtml : function(value) {
		return this.setAttr("innerHTML", value);
	},

	/**
	 * @param {} index
	 * @return {}
	 */
	getHtml : function(/* @default 0 */index) {
		var _e = this.elements[index || 0];
		return !!_e ? _e.innerHTML : null;
	},

	/**
	 * @param {string} value
	 */
	setVal : function(value) {
		if (QZFL.object.getType(value) == "array") {
			var _v = "\x00" + value.join("\x00") + "\x00";
			this.each(function(el) {
				if (/radio|checkbox/.test(el.type)) {
					el.checked = el.nodeType && ("\x00" + _v.indexOf(el.value.toString() + "\x00") > -1 || _v.indexOf("\x00" + el.name.toString() + "\x00") > -1);
				} else if (el.tagName == "SELECT") {
					//el.selectedIndex = -1;
					QZFL.object.each(el.options, function(e) {
						e.selected = e.nodeType == 1 && ("\x00" + _v.indexOf(e.value.toString() + "\x00") > -1 || _v.indexOf("\x00" + e.text.toString() + "\x00") > -1);
					});
				} else {
					el.value = value;
				}
			})

		} else {
			this.setAttr("value", value);
		}
		return this;
	},

	/**
	 * @param {} index
	 * @return {}
	 */
	getVal : function(/* @default 0 */index) {
		var _e = this.elements[index || 0],_v;

		if (_e) {
			if (_e.tagName == "SELECT"){
				_v = [];
				if (_e.selectedIndex<0) {
					return null;
				}

				//濡傛灉鏄崟閫夋
				if (_e.type == "select-one") {
					_v.push(_e.value);
				}else{
					QZFL.object.each(_e.options,function(e){
						if (e.nodeType == 1 && e.selected) {
							_v.push(e.value);
						}
					});
				}
			}else{
				_v = _e.value;
			}
		} else {
			return null
		}
		return _v;
	},

	/**
	 * @param {} className
	 */
	hasClass : function(className) {
		if(this.elements && this.elements.length){
			return QZFL.css.hasClassName(this.elements[0], className);
		}
		return false;
	},

	/**
	 * @param {} className
	 */
	addClass : function(className) {
		return this.each(function(el) {
			QZFL.css.addClassName(el, className);
		})
	},

	/**
	 * @param {} className
	 */
	removeClass : function(className) {
		return this.each(function(el) {
			QZFL.css.removeClassName(el, className);
		})
	},

	/**
	 * @param {} className
	 */
	toggleClass : function(className) {
		return this.each(function(el) {
			QZFL.css.toggleClassName(el, className);
		})
	},

	/**
	 * @param {} index
	 * @return {}
	 */
	getSize : function(/* @default 0 */index) {
		var _e = this.elements[index || 0];
		return !!_e ? QZFL.dom.getSize(_e) : null;
	},

	/**
	 * @param {} index
	 * @return {}
	 */
	getXY : function(/* @default 0 */index) {
		var _e = this.elements[index || 0];
		return !!_e ? QZFL.dom.getXY(_e) : null;
	},

	/**
	 * @param {} width
	 * @param {} height
	 */
	setSize : function(width, height) {
		return this.each(function(el) {
			QZFL.dom.setSize(el, width, height);
		})
	},

	/**
	 * @param {} X
	 * @param {} Y
	 */
	setXY : function(X, Y) {
		return this.each(function(el) {
			QZFL.dom.setXY(el, X, Y);
		})
	},

	/**
	 *
	 */
	hide : function() {
		return this.each(function(el) {
			QZFL.dom.setStyle(el, "display", "none");
		})
	},

	/**
	 *
	 */
	show : function(isBlock) {
		return this.each(function(el) {
			QZFL.dom.setStyle(el, "display", isBlock?'block':'');
		})
	},

	/**
	 * @param {} key
	 * @return {}
	 */
	getStyle : function(key, index) {
		var _e = this.elements[index || 0];
		return !!_e ? QZFL.dom.getStyle(_e, key) : null;
	},

	/**
	 * @param {} key
	 * @param {} value
	 */
	setStyle : function(key, value) {
		return this.each(function(el) {
			QZFL.dom.setStyle(el, key, value);
		})
	},
	/**
	 * 璁剧疆dom鐨勫睘鎬?	 *
	 * @param {string} key 灞炴€у悕绉?	 * @param {string} value 灞炴€?	 */
	setAttr : function(key, value) {
		key = (key=="class"?"className":key);

		return this.each(function(el) {
			el[key] = value;
		});
	},

	/**
	 * 鑾峰彇dom瀵硅薄鐨勫睘鎬?	 */
	getAttr : function(key, index) {
		key = key == "class" ? "className" : key;
		var node = this.elements[index || 0];
		return node ? (node[key] === undefined ? node.getAttribute(key) : node[key]) : null;
	}
});

// Extend Element relation
QZFL.element.extendFn(
/** @lends QZFL.ElementHandler.prototype */
{
	/**
	 * @return {}
	 */
	getPrev : function() {
		var _arr = [];
		this.each(function(el) {
			var _e = QZFL.dom.getPreviousSibling(el);
			//if (_e) {
				_arr.push(_e);
			//}
		});

		return QZFL.element.get(_arr);
	},

	/**
	 * @return {}
	 */
	getNext : function() {
		var _arr = [];
		this.each(function(el) {
			var _e = QZFL.dom.getNextSibling(el);
			//if (_e) {
				_arr.push(_e);
			//}
		});

		return QZFL.element.get(_arr);
	},

	/**
	 * @return {}
	 */
	getChildren : function() {
		var _arr = [];
		this.each(function(el) {
			var node = QZFL.dom.getFirstChild(el);
			while (node) {
				if (!!node && node.nodeType == 1) {
					_arr.push(node);
				}
				node = node.nextSibling;
			}
		});

		return QZFL.element.get(_arr);
	},

	/**
	 * @return {}
	 */
	getParent : function() {
		var _arr = [];
		this.each(function(el) {
			var _e = el.parentNode;
			//if (_e) {
				_arr.push(_e);
			//}
		});

		return QZFL.element.get(_arr);
	}
});

// Extend
QZFL.element.extendFn(
/** @lends QZFL.ElementHandler.prototype */
{

	/**
	 * @param {} tagName
	 * @param {} attributes
	 * @return {}
	 */
	create : function(tagName, attributes) {
		var _arr = [];
		this.each(function(el) {
			_arr.push(QZFL.dom.createElementIn(tagName, el, false, attributes));
		});
		return QZFL.element.get(_arr);
	},

	/**
	 * @param {} el
	 */
	appendTo : function(el) {
		var el = (el.elements && el.elements[0]) || QZFL.dom.get(el);
		return this.each(function(element) {
			el.appendChild(element)
		});
	},

	/**
	 * @param {} el
	 */
	insertAfter : function(el) {
		var el = (el.elements && el.elements[0]) || QZFL.dom.get(el), _ns = el.nextSibling, _p = el.parentNode;
		return this.each(function(element) {
			_p[!_ns ? "appendChild" : "insertBefore"](element, _ns);
		});

	},

	/**
	 * @param {} el
	 */
	insertBefore : function(el) {
		var el = (el.elements && el.elements[0]) || QZFL.dom.get(el), _p = el.parentNode;
		return this.each(function(element) {
			_p.insertBefore(element, el)
		});
	},

	/**
	 *
	 */
	remove : function() {
		return this.each(function(el) {
			QZFL.dom.removeElement(el);
		})
	}
});




/////////////
//effect.js
/////////////


/**
 * QZFL鍩虹鍔ㄧ敾绫? * @author joltwang
 * @version 1.0.0.1
 * @date 2013.3.12 
 * @edit by hank
 * @example 鏇村绀轰緥璇风湅src鐩綍閲岀殑examples
 */
QZFL.effect = {
	/**
      * 鍔ㄧ敾鍩虹鏂规硶锛屽浼犲叆鐨勫css灞炴€у€艰繘琛屽姩鐢诲彉鍖栵紝瀵逛簬鏀寔css3鐨勬祻瑙堝櫒閲囩敤css3 transform锛屽浜庝笉鏀寔css3鐨勫叾浠栨祻瑙堝櫒閲囩敤timer璁＄畻鍏抽敭甯?      * 鏉ユ敼鍙樺厓绱燾ss灞炴€у€笺€?      * @param elem [string or object] 闇€瑕佸姩鐢诲鐞嗙殑dom
      * @param prop [object] 浼犲叆闇€瑕佷慨鏀圭殑css灞炴€э紝濡倇opacity: 0.25,left: '+=50',width: '+=150',height: '+=100'}
      * @param opts [object] 閰嶇疆淇℃伅锛屽鍔ㄧ敾鎵ц鏃堕棿锛屾墽琛屽畬鍥炶皟鍑芥暟{duration:1000, complete:callbackFun}
      * @example 
      * QZFL.effect.run($('demo'), {
    	  opacity: 0,
    	  top:'100',
          width: '+=150',
          height: '+=100'
        }, {
        	duration : 1000,
            complete : function(){},
            change : QZFL.emptyFn,
            start : QZFL.emptyFn
      });
      */
    off : 0,  //鏄惁鍏抽棴鍔ㄧ敾
    mode : [], //妯″紡 css3 or ''
    init : function(){  //鍒濆鍖栦簨浠剁被鍨嬬殑娴忚鍣ㄦ敮鎸佸害
        var classArray = [
            ['webkit', 'WebkitTransition'],
            ['firefox', 'MozTransition'],
            ['opera', 'OTransition'],
            ['ie', 'msTransition']
        ],
        ua = QZFL.userAgent, agent = '', cName = '';
        //鍐呭瓨鍥炴敹鍚?涓€娆℃€у嚱鏁?        for(var i = 0, len = classArray.length; i < len; i++){
            if(ua[classArray[i][0]]){
                agent = classArray[i][0];
                cName = classArray[i][1];
                break;
            }
        }
        return QZFL.effect.mode = ((cName in document.documentElement.style) ?
            [agent, 'css3'] : 
            [agent]);
            //浠呬竴涓彉閲忓氨濂戒簡isCSSTransition
    },
	run : function(elem, prop, opts){
        var o = QZFL.effect,
            tid = ++o._uniqueID,
            fpropArray,
            fprop, qDom;
            
        if(!elem){
            return;
        }
        if(!o.mode[0]){ //浠呭垵濮嬪寲涓€娆?            o.init();
        }
		opts = o._opt(opts);
		opts.start();//鎵ц寮€濮嬫椂鍥炶皟
		
		elem = QZFL.dom.get(elem);
		//瑙ｆ瀽灞炴€?		fpropArray = o._prop(prop,elem);
		fprop = fpropArray[0];
		
		elem._tid = tid;
		
		if(o.off){
		    qDom = QZFL.dom;
		    for(var i in fpropArray[1]){
		        qDom.setStyle(elem, i, fpropArray[1][i]);
		    }
		    window.setTimeout(opts.complete, opts.duration);
		}else{
		    var t = new QZFL.tweenMaker(0, 100, opts.duration, opts.interval, opts);
            //瀵逛簬绾痗ss3鍔ㄧ敾锛屼笖娌℃湁change鍥炶皟鐨勶紝鏄笉鏄笉鐢╰weenMaker?
            t.onStart = (o.mode[1]=='css3') ? function(){
                o._tweenArray[tid] = t;
                (new QZFL.cssTransfrom(elem,fprop,opts)).firecss();
            } : function(){
                o._tweenArray[tid] = t;
            };
            t.onChange = (o.mode[1]!='css3') ? function(p){ //涓嶇敤姣忔杩愯鏃舵潵鍒ゆ柇mode
                o.drawStyle(fprop,p,elem);
                opts.change(p);
            } : function(p){
                opts.change(p);
            };
            
            t.onEnd = function(){
                if(o.mode[1]!='css3'){
                    opts.complete();
                }
                delete o._tweenArray[elem._tid];
            };
            //TODO 杩欎釜鍦版柟鏀规垚鍔犲埌涓€涓槦鍒楅噷闈㈠幓
            t.start();
		}
 	},
 	/**
 	 * 鑾峰彇鍔ㄧ敾杩涜鐨勭櫨鍒嗘瘮
 	 * @param [object] elem 鍔ㄧ敾鍏宠仈鐨勫厓绱? 	 */
 	getPercent : function(elem){
 		var elem = QZFL.dom.get(elem), tid = elem._tid,t = QZFL.effect._tweenArray[tid];
 		return t.getPercent();
 	},
 	
 	/**
 	 * 鍋滄鍔ㄧ敾,鐩存帴鍒版渶缁堢姸鎬? 	 * @param [object] elem 鍔ㄧ敾鍏宠仈鐨勫厓绱? 	 */
 	stop: function(elem){
 		var es = QZFL.effect, webkit = (es.mode[1]=='css3'), o;
 		elem = QZFL.dom.get(elem);
 		if(webkit){
 			(o = elem._transition) && o.stop();   
 		}else{
			var tid = elem._tid,t = es._tweenArray[tid];
			t && t.stop();
 		}
 		return es;
	},
 	
 	drawStyle : function(prop, p, elem){
 	    var DOM = QZFL.dom, tmp, cssText = '', re, S = QZFL.string;
 		p*=0.01;
    	QZFL.object.each(prop,function(f, pname){
    		var s = f.start, e = f.end, u = f.unit;
    		v = e>=s?((e-s)*p+s):(s-(s-e)*p);
    		re = DOM.convertStyle(elem, pname, v+u);
    		cssText += (S.reCamelCase(re.prop) + ':' + re.value + ';');
        });
        //缁熶竴甯х殑澶勭悊
        elem.style.cssText += (';' + cssText);
    },
 	
 	_tweenArray : {},
    _uniqueID: 0,
	
	_opt : function(opts){
		var opt = opts,o = QZFL.effect;
		opt.duration = opts.duration || 500;
		opt.easing = opts.easing || 'ease';
		opt.complete = opts.complete || QZFL.emptyFn;
		opt.interval = opts.interval || 16;
		opt.start = opts.start || QZFL.emptyFn;
		opt.change = opts.change || QZFL.emptyFn;
		
		return opt;
	},
	
	_prop : function(prop,elem){
        var fprop = {},es = QZFL.effect,webkit = (es.mode[1]=='css3'),endCSSMap = {};
        //閬嶅巻姣忎釜灞炴€?        QZFL.object.each(prop, function(val, pname){
        	pname = QZFL.string.camelCase(pname);
        	if(QZFL.object.getType(val) == "object"){
        		var f = es._cssValue(elem,val.value,pname);
        		endCSSMap[pname] = (val.value = f.end + (f.unit?f.unit:0));
        		if(webkit){
        			fprop[pname] = val;
        		}else{
        			fprop[pname] = f;
        		}
        	}else{
        		var d = es._cssValue(elem,val,pname),tmp;
        		endCSSMap[pname] = (tmp = d.end + (d.unit?d.unit:0));
        		if(webkit){
        			d =  tmp; 
        		}
        		fprop[pname] = d;
        	}
        });
        
        return [fprop, endCSSMap];
    },
    
    /**
     * 璁＄畻鏌愪釜鍔ㄧ敾鍏冪礌涓婄殑璧峰鍊煎強鍗曚綅
     */
    _cssValue : function(elem, val, name){
		var fnum = /^([+\-]=)?([\d+.\-]+)([a-z%]*)$/i,
			fprop = {},
			parts = fnum.exec(val+''),
			o = QZFL.effect,
			start = o._cur(elem,name);//淇寮€濮嬫椂鐨勪綅缃?
        if(parts){ //濡傛灉鏄?= or -=
            var end = parseFloat(parts[2]),
            	unit = parts[3]||(o._cssNumber[name] ? "" : "px");
            	
            if(unit !== "px"){//鍗曚綅涓嶆槸px鐨?                QZFL.dom.setStyle(elem, name, (end || 1) + unit);
                start = ((end || 1) / o._cur(elem,name)) * start;
                QZFL.dom.setStyle(elem, name, start + unit);
            }
            if(parts[1]){
                end = ((parts[1] === "-=" ? -1 : 1) * end) + start;
            }
            fprop = {start : start,end : end,unit:unit};
        }else{
            fprop = {start : start,end : val,unit:''};
        }
        
        return fprop;
    },
    
    _cssNumber : {"zIndex": true,"fontWeight": true,"opacity": true,"zoom": true,"lineHeight": true},
	
	_cur : function(elem,p) {
		var parsed, r = QZFL.dom.getStyle(elem,p);
		if(elem!=null && elem[p] != null && (!elem.style || elem.style[p] == null)){
			return elem[p];
		}
		return isNaN(parsed = parseFloat(r)) ? !r || r === "auto" ? 0 : r : parsed;
	},
	
	show : function(elem, opts, cb){
        var d = QZFL.dom, duration;
        elem = d.get(elem);
        opts = opts || {};
        duration = ((typeof(opts) == 'number') ? opts : opts.duration);
        cb = opts.start || QZFL.emptyFn;
        QZFL.effect.run(elem, {
          opacity: 1
        }, QZFL.object.extend(opts, {
            duration : duration || 1000,
            start : function(){
                d.setStyle(elem,'opacity',0);
                d.setStyle(elem,'display','');
                cb();
            },
	    complete : cb
        }));
    },
    
    hide : function(elem, opts, cb){
        var d = QZFL.dom, cb, duration;
        elem = d.get(elem);
        opts = opts || {};
        duration = ((typeof(opts) == 'number') ? opts : opts.duration);
        cb = cb || opts.complete || QZFL.emptyFn;
        QZFL.effect.run(elem, {
          opacity: 0
        }, QZFL.object.extend(opts, {
            duration : duration || 1000,
            complete : function(){
                d.setStyle(elem,'display','none');
                d.setStyle(elem,'opacity',1);
                cb();
            }
        }));
    },
    /**
     * 鏄剧ず鍜岄殣钘忓垏鎹㈠嚱鏁?     * @param {Function} [opts.onShow] 灞曠ず鐨勫洖璋?     * @param {Function} [opts.onHode] 娑堝け鐨勫洖璋?     */
    toggle : function(elem, opts){
        var o = QZFL.effect;
        opts = opts || {};
        if(o._isHidden(elem)){
            o.stop(elem).show(elem, opts);
        }else{
            o.stop(elem).hide(elem, opts);
        }
    },
    slideDown : function(elem, opts){
        var d = QZFL.dom,
            attrs,
            o = QZFL.effect,
            toValue = {},
            _obj = QZFL.object,
            duration, start, complete;
        elem = d.get(elem);
        attrs = o._checkVerticalStyle(elem, {status:'down'});
        opts = opts || {};
        
        if(attrs){
            duration = ((typeof(opts) == 'number') ? opts : opts.duration);
            start = opts.start || QZFL.emptyFn;
            complete = opts.complete || QZFL.emptyFn;
            o.run(elem, attrs, _obj.extend(opts, {
                duration : duration||1000,
                start : function(){
                    if(attrs && !opts.noClear){
                        _obj.each(attrs, function(v, i){
                            d.setStyle(elem, i, '0px');
                        });
                    }
                    d.setStyle(elem,'display','');
                    start();
                },
                complete : function(){
                    complete();
                    //娓呴櫎鐘舵€?                    o._checkVerticalStyle(elem, {clear:1}); 
                }
            }));
        }
    },
    slideUp : function(elem, opts){
        var d = QZFL.dom,
            attrs,
            o = QZFL.effect,
            toValue = {},
            _obj = QZFL.object,
            duration,
            complete;
        elem = d.get(elem);
        
        attrs = o._checkVerticalStyle(elem, {status:'up'});
        if(attrs){
            _obj.each(attrs, function(v, i){
                toValue[i] = '0px';
            });
            opts = opts || {};
            duration = ((typeof(opts) == 'number') ? opts : opts.duration);
            complete = opts.complete || QZFL.emptyFn;
            //灞曠ず
            d.setStyle(elem,'display','');
            o.run(elem, toValue, _obj.extend(opts, {
                duration : duration||1000,
                complete : function(){
                    d.setStyle(elem,'display','none');
                    //鐒跺悗杩樺師height padding margin
                    _obj.each(attrs, function(v, i){
                        d.setStyle(elem, i, v); 
                    });
                    //娓呴櫎toggle鐘舵€?                    o._checkVerticalStyle(elem, {clear:1});
                    complete();
                }
            }));
        }
    },
    _slideArray : {},
    //TODO 杩欓噷鍒囨崲鐨勮瘽鎬庝箞澶勭悊锛屾瘮濡俿lideUp鍒颁竴鍗婅slideDown锛屽張瑕佽€冭檻涓€旀敼鍙樼殑鍊?涓嶇敤鑰冭檻)
    //鍦ㄨ繍琛岃繃绋嬩腑璁板綍濂界姸鎬侊紝杩愯瀹屾垚娓呮帀锛?    slideToggle : function(elem, opts){
        var o = QZFL.effect, status, opts = opts || {};
        if(o._isHidden(elem)){ //鍒濆鐘舵€侊紝鍐冲畾瑕佸睍寮€杩樻槸鎷夎捣
            o.stop(elem).slideDown(elem, opts);
        }else{
            if(elem._slideid){ //濡傛灉鍦╰oggle杩愬姩涓?                status = o._slideArray[elem._slideid].status;
                if(status == 'up'){
                    opts.noClear = 1;
                    o._slideArray[elem._slideid].status = 'down';
                    o.stop(elem).slideDown(elem, opts);
                }else if(status == 'down'){
                    o._slideArray[elem._slideid].status = 'up';
                    o.stop(elem).slideUp(elem, opts);
                }
            }else{
                o.stop(elem).slideUp(elem, opts);
            }
        }
    },
    /**
     * 妫€鏌ュ瀭鐩寸殑灞炴€э紝鍖呮嫭'marginTop', 'marginBottom', 'paddingTop', 'paddingBottom', 'height'
     */
    _checkVerticalStyle : function(elem, opts){
        var name = ['marginTop', 'marginBottom', 'paddingTop', 'paddingBottom', 'height'],
            obj = {},
            D = QZFL.dom, re, o = QZFL.effect;
        if(opts.clear){ //娓呴櫎鏁版嵁
            if(o._slideArray[elem._slideid]){
                delete o._slideArray[elem._slideid];
                delete elem._slideid;
            }
            return null;
        }
        if(!elem._slideid){
            elem._slideid = ++o._uniqueID;
        }
        if(!(re = o._slideArray[elem._slideid])){
            //褰撶珫鍚戠殑灞炴€ч兘涓嶄负0鏄墠鍙?            QZFL.object.each(name, function(v){
                 var val = parseInt(D.getStyle(elem, v), 10);
                 if(val){
                     obj[v] = val;
                     re = 1;
                 }
            });
            o._slideArray[elem._slideid] = {
                pps : obj,
                status : opts.status
            };
        }else{
            obj = re.pps;
        }
        //濡傛灉_slideArray涓湁elem鐨勭紦瀛樻暟鎹紝閭ｄ箞杩斿洖缂撳瓨鎴栬€呮暟鎹?        return re ? obj : null;
    },
    _isHidden : function(elem){
        return QZFL.dom.getStyle(elem, 'display') == 'none';
    }
};

/**
 * 鍒堕€犲姩鐢荤殑绫? * @param [number] startvalue 寮€濮嬪€? * @param [number] endvalue 缁撴潫鍊? * @param [float] duration 鎸佺画鏃堕棿
 * @param [number] interval 姣忎釜鏃堕棿鐗囩殑鏃堕棿
 * @param [object] opt 鍙紶鍏unctor鏉ヤ綔涓虹畻瀛? */
QZFL.tweenMaker = function(startvalue,endvalue,duration,interval,opt){
	var o = this, opt = opt || {}, easing;
    o.duration = duration || 500;
    o.interval = interval || 16;
    o.startValue = startvalue;
    o.endValue = endvalue;
//  o.count = Math.ceil(o.duration/o.interval); //鏃犵敤鍙傛暟
//  o.elapse = 0; //鏃犵敤鍙傛暟
    easing = opt.easing || 'ease';
    o.functor = (typeof(opt.functor) == 'function' ? opt.functor : (o.functors[easing] || o.functors['ease']));

    o.onStart = o.onChange = o.onEnd = QZFL.emptyFn;
    o.playing = false;

    o.changeValue = o.endValue - o.startValue;
	o.currentValue = 0;
	/**鍔ㄧ敾鍣?	var proto = QZFL.tweenMaker.prototype;
	if(!proto.animator){
	    proto.animator = new Animator();
	}*/
};

QZFL.tweenMaker.prototype = {
    functors : {
        //time, startValue, changeValue, duration
        //default锛屽洜涓篶ss3涓槸涓嶅绉扮殑鍙橀€燂紝鏆傛椂鍙兘鐢ㄨ繖涓ā鎷?        'ease' : function(t, s, c, d){ //3娆℃柟鐨勭紦鍔?            if ((t/=d/2) < 1) return c/2*t*t*t + s;
            return c/2*((t-=2)*t*t + 2) + s;
        },
        'linear' : function(t, s, c, d){
            return c * t / d + s;
        },
        'ease-in' : function(t, s, c, d){
            return c*(t/=d)*t*t + s;
        },
        'ease-out' : function(t, s, c, d){
            return c*((t=t/d-1)*t*t + 1) + s;
        },
        'ease-in-out' : function(t, s, c, d){
            if ((t/=d/2) < 1) return c/2*t*t*t + s;
            return c/2*((t-=2)*t*t + 2) + s;
        }
    },
	//寮€濮嬪姩鐢?    start : function(){
    	var o = this/*, a = this.animator*/;
        o.playing = true;
        o._startTime = new Date().getTime();
        o.onStart.apply(o);
        o._runTimer();
    },

    _runTimer : function(){
        var o = this;
        if(o.playing){
            o._playTimer();
            setTimeout(function(){
                o._runTimer.apply(o,[]);
            }, o.interval);
        }
    },

    _playTimer: function (time) {
        var _end = false, o = this, time = (new Date().getTime() - o._startTime);
        if (time > o.duration) {
            time = o.duration;
            _end = true;
        }
        o.currentValue = o.functor(time, o.startValue, o.changeValue, o.duration);
        o.onChange.apply(o, [o.getPercent()]);
        // 鍒ゆ柇鏄惁鎾斁缁撴潫
        if (_end) {
            o.playing = false;
            o.onEnd.apply(this);

            // 鎾斁瀹屾垚寮鸿揩IE鍥炴敹鍐呭瓨
            if (window.CollectGarbage){
                CollectGarbage();
            }
        }
    },

    stop : function(){
        this.playing = false;
//      this.currentValue = this.endValue;
    },
	
	//鑾峰彇鐧惧垎姣?    getPercent : function(){
        return (this.currentValue - this.startValue)/this.changeValue * 100;
    }
};
/**
 * css3 transfrom[css3妯″紡]锛屼緵QZFL.effect浣跨敤锛屼笉鍗曠嫭璋冪敤
 */
QZFL.cssTransfrom = function(elem, prop, opts){
	var o = this;
	o._elem = elem;
	//杩欓噷_uid瑕佷繚鎸佸敮涓€鎬э紝涓嶈兘鐢≦ZFL.now()锛屼袱娆¤皟鐢ㄤ細寰楀埌鍚屾牱鐨勫€?	o._uid = 'uid_'+(++QZFL.cssTransfrom._count);
	if(!o._running && prop){
	    o._conf = prop;
	    o._duration = ('duration' in opts) ? opts.duration/1000: 0.5;
	    o._delay = ('delay' in opts) ? opts.delay: 0;
	    o._easing = opts.easing || 'ease';
	    o._count = 0;
	    o._running = false;
	    o._callback = QZFL.lang.isFunction(opts.complete)?opts.complete:QZFL.emptyFn;
	    o._change = opts.change;
	    elem._transition = o;
	}
	return o;
};
QZFL.cssTransfrom._cssText = {};
QZFL.cssTransfrom._attrs = {};
QZFL.cssTransfrom._hasEnd = {};
QZFL.cssTransfrom._count = 10000;
QZFL.cssTransfrom.prototype = {
    init : function(){
        var map = [
            ['webkit', '-webkit-transition', 'WebkitTransition', 'webkitTransitionEnd', 'WebkitTransform'],
            ['firefox', '-moz-transition', 'MozTransition', 'transitionend', 'MozTransform'],
            ['opera', '-o-transition', 'OTransition', 'oTransitionEnd', 'OTransform'],
            ['ie', '-ms-transition', 'msTransition', 'MSTransitionEnd', 'msTransform']
        ],
        tiClassPrefix, //ti: transition  tf: transform
        tiStyleName,
        tiEvtName,
        tfStyleName,
        ua = QZFL.userAgent,
        proto;
        //鍐呭瓨鍥炴敹鍚?涓€娆℃€у嚱鏁?        for(var i = 0, len = map.length; i < len; i++){
            if(ua[map[i][0]]){
                tiClassPrefix = map[i][1];
                tiStyleName = map[i][2];
                tiEvtName = map[i][3];
                tfStyleName = map[i][4];
                break;
            }
        }
        proto = QZFL.cssTransfrom.prototype;
        proto.TRANSITION  = {
            'classPrefix' : tiClassPrefix,
            'event' : tiEvtName,
            'styleName' : tiStyleName
        };
        proto.TRANSFORM  = {
            'styleName' : tfStyleName
        };
    },
    firecss : function(){
        var o = this, elem = o._elem,uid = o._uid,
            conf = this._conf,
            getStyle = QZFL.dom.getStyle,
            ct = QZFL.cssTransfrom,
            attrs;
        var cssTextArray = [],
            delayKey,
            delayVal = [],
            pptVal = [],
            durationKey,
            durationVal = [],
            easingKey,
            easingVal = [],
            TRANSFORM,
            cPrefix,
			_cprop='',
			cssText;
        //鍒濆鍖栦簨浠剁被鍨?        if(!(this.TRANSITION && this.TRANSITION.classPrefix)){
            this.init();
        }
        this._running = true;
        cPrefix = this.TRANSITION.classPrefix;
        delayKey = cPrefix+'-delay';
        this.pptKey = cPrefix+'-property';
        durationKey = cPrefix+'-duration';
        easingKey = cPrefix+'-timing-function';
        TRANSFORM = this.TRANSFORM.styleName; //transform鍔ㄧ敾
        
        if (conf.transform && !conf[TRANSFORM]) {
            conf[TRANSFORM] = conf.transform;
            delete conf.transform;
        }
        for(var attr in conf) {
            if(conf.hasOwnProperty(attr)){
                o._addProperty(attr,conf[attr]);
                if(elem.style[attr] === '') {
                	QZFL.dom.setStyle(elem, attr, getStyle(elem, attr));
                }
            }
            _cprop = attr;
        }
        
        pptVal.push(getStyle(elem, this.pptKey));
        if(pptVal[0] && (pptVal[0] !== 'all')){
            durationVal.push(getStyle(elem, durationKey));
            easingVal.push(getStyle(elem, easingKey));
            delayVal.push(getStyle(elem, delayKey));
        }else{
            pptVal.pop();
        }
        attrs = ct._attrs[uid];
        for(var name in attrs){
            hyphy = o._toHyphen(name);
            attr = attrs[name];
            if (attrs.hasOwnProperty(name) && attr.transition === o) {
                if (name in elem.style) {
                    durationVal.push(parseFloat(attr.duration)+'s');
                    delayVal.push(parseFloat(attr.delay)+'s');
                    easingVal.push(attr.easing);
                    pptVal.push(hyphy);
                    cssTextArray.push(hyphy + ': ' + attr.value);
                } else {
                    o._removeProperty(name);
                }
            }
        }
        
        if(!ct._hasEnd[uid]) {//鍒ゆ柇鏄惁缁撴潫
            elem.addEventListener(this.TRANSITION.event, (elem._transitionCb = function(e){
                o._onTransfromEnd(e,uid);
            }), false);
            //鍐嶅姞涓€閲嶄繚闄?            o.timer = window.setTimeout(function(){
                o._end();
            }, o._duration * 1000);
            ct._hasEnd[uid] = true;
        }
		cssText = cssTextArray.join(';');
		ct._cssText[uid] = {};
		ct._cssText[uid].property = pptVal;
		ct._cssText[uid].style = elem.style.cssText; //璁板綍鍘熷style
        elem.style.cssText += ['',this.pptKey + ':' + pptVal.join(','),
            durationKey + ':' + durationVal.join(','),
            easingKey + ':' + easingVal.join(','),
            delayKey + ':' + delayVal.join(','),
            cssText, ''].join(';');//寮€濮嬫覆鏌揷ss3锛屽吋瀹规病鏈変互锛涚粨鏉熺殑鎯呭喌
    },

	_toHyphen : function(prop) {
        prop = prop.replace(/([A-Z]?)([a-z]+)([A-Z]?)/g, function(m0, m1, m2, m3) {
            var str = '';
            m1&&(str += '-' + m1.toLowerCase());
            str += m2;
            m3&&(str += '-' + m3.toLowerCase());
            return str;
        });
        return prop;
	},

    _endTransfrom: function(sname) {//缁撴潫鍚庯紝瀵瑰厓绱犱笂鐨剆tyle鍋氫笅娓呯悊
        var QF = QZFL,
            elem = this._elem,
            pptKey = this.pptKey,
            value = QF.dom.getStyle(elem, pptKey);
        
        if(!value){ //鍏煎opera
            pptKey = QF.string.camelCase(pptKey);
            value = elem.style[pptKey];
        }
        
        if(typeof value === 'string'){
            value = value.replace(new RegExp('(?:^|,\\s)' + sname + ',?'), ',');
            value = value.replace(/^,|,$/, '');
            elem.style[this.TRANSITION.styleName] = value || null; //clear transition property
        }
    },

    _onTransfromEnd: function(e,uid){
        var pname = QZFL.string.camelCase(e.propertyName),
            elapsed = e.elapsedTime,
            attrs = QZFL.cssTransfrom._attrs[uid],
            attr = attrs&&attrs[pname],
            tran = (attr) ? attr.transition : null, _cprop='';
        
        if(tran){
            window.clearTimeout(this.timer);
            this.timer = null;
        	for(var attr in this._conf) {
	            _cprop = attr;
	        }
            tran._removeProperty(pname);
            tran._endTransfrom(pname);
            if(tran._count <= 0){
                tran._end(elapsed);
            }
        }
    },

    _addProperty: function(prop, conf){//瀵瑰姩鐢婚厤缃」璁＄畻涓€涓嬶紝骞舵坊鍔犲埌_attrs閲?        var o = this,node = this._elem,
            uid = o._uid,
            attrs = QZFL.cssTransfrom._attrs[uid],
            computed,compareVal,dur,attr,val;
        if(!attrs) {
            attrs = QZFL.cssTransfrom._attrs[uid] = {};
        }
        attr = attrs[prop];
        if(conf && conf.value !== undefined) {
            val = conf.value;
        }else if(conf !== undefined) {
            val = conf;
            conf = {};
        }
        if(attr && attr.transition){
            if(attr.transition !== o){
               attr.transition._count--;
            }
        }
        o._count++;
        dur = ((typeof conf.duration != 'undefined') ? conf.duration : o._duration) || 0.0001;
        attrs[prop] = {
            value: val,
            duration: dur,
            delay: (typeof conf.delay != 'undefined') ? conf.delay : o._delay,
            easing: conf.easing || o._easing,
            transition: o
        };
        computed = QZFL.dom.getStyle(node, prop);
        compareVal = (typeof val === 'string') ? computed : parseFloat(computed);
        //濡傛灉棰勬湡鍊煎拰鐜板湪鍊兼槸涓€鏍风殑
        if (compareVal === val) {
            setTimeout(function() {
                o._onTransfromEnd.call(o, {
                    propertyName: prop,
                    elapsedTime: dur
                }, uid);
            }, dur * 1000);
        }
    },

    _removeProperty: function(prop){  //娓呯悊鍙傛暟
        var o = this, attrs = QZFL.cssTransfrom._attrs[o._uid];
        if (attrs && attrs[prop]) {
            delete attrs[prop];
            o._count--;
        }
    },

    _end: function(){  //css3鍔ㄧ敾缁撴潫锛屾墽琛屽洖璋?        var o = this, elem = o._elem, callback = o._callback;
        o._running = false;
        o._callback = null;
        //bug:褰撳姩鐢昏繍琛岃繃绋嬩腑stop锛宑allback渚濈劧浼氭墽琛岋紝浜嬩欢绉婚櫎鏈夐棶棰橈紵
        if(elem&&callback&&!this._stoped){
        	setTimeout(function(){callback();},1);
        	//娓呴櫎鐜板満
        	o.clearStatus(elem);
       	}
    },
    stop : function(){
        var uid = this._uid, elem = this._elem, cText, pps, styleText = [];
        cText = QZFL.cssTransfrom._cssText[uid];
        pps = cText.property;
        for(var i = 0; i < pps.length; i++){
            styleText.push(pps[i] + ':' + QZFL.dom.getStyle(elem, pps[i]));
        }
        //bug:杩欐牱浼氬鑷村姩鐢昏繃绋嬩腑澧炲姞鐨勫睘鎬ц娓呮帀
//      styleText.length && (elem.style.cssText = cText.style + ';' + styleText.join(';'));  //绉婚櫎transition灞炴€?        this.clearStatus(elem, styleText.join(';'));
        this._stoped = true;
    },
    /**
     * 娓呴櫎涓€涓嬬幇鍦?     */
    clearStatus : function(elem, style){
        elem.style.cssText = elem.style.cssText.replace(/[^;]+?transition[^;]+?;/ig,'') + (style ? style : '');
        if(elem._transitionCb){
            elem.removeEventListener(this.TRANSITION.event, elem._transitionCb, false); //绉婚櫎浜嬩欢鐜板満
            elem._transitionCb = null;
        }
    }
 };
 
QZFL.now = function(){
    return (new Date()).getTime();
};

QZFL.string = QZFL.string||{};
QZFL.string.camelCase = function(s){
	var r = /-([a-z])/ig;
	return s.replace(r,function(all,letter) {
		return letter.toUpperCase();
	});
};
QZFL.string.reCamelCase = function(s){
    var r = /[A-Z]/g;
    return s.replace(r,function(all,letter) {
        return '-' + all.toLowerCase();
    });
};

/**
 * 鐢ㄦ柊鐨凲ZFL.effect瀵规帴涔嬪墠鐨凾ween鏂规硶,鑰佺殑鐗堟湰涓嶅啀鍚堝叆QZFL鐗堟湰閲岋紝濡傛兂浣跨敤璇峰崟鐙姞杞戒竴浠絫ween.js
 * @param {string} elem 闇€瑕佸姩鐢诲鐞嗙殑dom
 * @param {string} prop css灞炴€у悕
 * @param {function} func 鍔ㄧ敾绠楀瓙
 * @param {string or number} startValue 鍒濆鍊? * @param {string or number} finishValue 缁撴潫鍊? * @param {number} duration 鍔ㄧ敾鎵ц鏃堕棿
 */
QZFL.Tween = function(elem, prop, func, startValue, finishValue, duration){
	var o = this;
	o.elem = QZFL.dom.get(elem);
	o.prop = {};
	o.sv = startValue;
	o.fv = finishValue;
	o.pname = prop;
	o.prop[prop] = parseInt(finishValue);
	o.opts = {duration : duration*1000};
	o.onMotionStart = QZFL.emptyFn;
	o.onMotionChange = null;
	o.onMotionStop = QZFL.emptyFn;
	o.css = true;
};

/**
 * 寮€濮嬭繍琛屽姩鐢? */
QZFL.Tween.prototype.start = function(){
	var o = this,s = parseInt(o.sv),e = parseInt(o.fv);
	var set = QZFL.dom.setStyle(o.elem,o.pname,o.sv);
	if(set){
		o.opts.complete = o.onMotionStop;
		o.opts.change = function(p){
			p*=0.01;
			var v = e>=s?((e-s)*p+s):(s-(s-e)*p);
			o.onMotionChange&&o.onMotionChange.apply(o,[o.elem,o.pname,v]);
		}
		o.onMotionStart.apply(o);
		QZFL.effect.run(o.elem,o.prop,o.opts);
	}else{
		o.css = false;
		var t = new QZFL.tweenMaker(s,e,o.opts.duration,o.opts.interval||15);
		t.onStart = function(){
			o.t = t;
			o.onMotionStart.apply(o);
		};
		t.onChange = function(p){
			p*=0.01;
			var v = e>=s?((e-s)*p+s):(s-(s-e)*p);
			o.onMotionChange&&o.onMotionChange.apply(o,[o.elem,o.pname,v]);
		};
		t.onEnd = function(){
			o.onMotionStop.apply(o);
		};
		t.start();
	}
};

//鑾峰彇鍔ㄧ敾杩涘害鐧惧垎姣?QZFL.Tween.prototype.getPercent = function(){
	return this.css ? QZFL.effect.getPercent(this.elem):this.t.getPercent();
};

/**
 * 鍋滄or鏆傚仠鍔ㄧ敾
 */
QZFL.Tween.prototype.stop = function() {
	QZFL.effect.stop(this.elem);
};

/**
 * 鍏煎鑰佺殑绠楀瓙绫伙紝鑰佺殑鐗堟湰涓嶅啀鍚堝叆QZFL鐗堟湰閲岋紝濡傛兂浣跨敤璇峰崟鐙姞杞戒竴浠絫ween.js
 */
QZFL.transitions = {};



/////////////
//tween_extend.js
/////////////


/**
 * @fileoverview 鎶妕ween绫荤殑鎺ュ彛灏佽鍒癚ZFL Elements
 * @version 1.$Rev: 1723 $
 * @author QzoneWebGroup
 * @lastUpdate $Date: 2010-04-08 19:26:57 +0800 (鍛ㄥ洓, 08 鍥涙湀 2010) $
 */
;(function() {
	/**
	 * resize鍜宮ove鐨勭畻娉?	 */
	var _easeAnimate = function(_t, a1, a2, ease) {
		var _s = QZFL.dom["get" + _t](this), _reset = typeof a1 != "number" && typeof a2 != "number";

		if (_t == "Size" && _reset) {
			QZFL.dom["set" + _t](this, a1, a2);
			var _s1 = QZFL.dom["get" + _t](this);
			a1 = _s1[0];
			a2 = _s1[1];
		}

		var _v1 = _s[0] - a1;
		var _v2 = _s[1] - a2;

		var n = new QZFL.Tween(this, "_p", QZFL.transitions[ease] || QZFL.transitions.regularEaseOut, 0, 100, 0.5);

		n.onMotionChange = QZFL.event.bind(this, function() {
			var _p = arguments[2];
			QZFL.dom["set" + _t](this, typeof a1 != "number" ? _s[0] : (_s[0] - _p / 100 * _v1), typeof a2 != "number" ? _s[1] : (_s[1] - _p / 100 * _v2));
		});

		// reset size to auto
		if (_t == "Size" && _reset) {
			n.onMotionStop = QZFL.event.bind(this, function() {

				QZFL.dom["set" + _t](this);

			});
		}

		n.start();
	};

	var _easeShowAnimate = function(_t, ease) {
		var n = new QZFL.Tween(this, "opacity", QZFL.transitions[ease] || QZFL.transitions.regularEaseOut, (_t ? 0 : 1), (_t ? 1 : 0), 0.5);
		n[_t ? "onMotionStart" : "onMotionStop"] = QZFL.event.bind(this, function() {
			this.style.display = _t ? "" : "none";
			QZFL.dom.setStyle(this, "opacity", 1);
		});
		n.start();
	};

	var _easeScroll = function(top, left, ease) {
		if (this.nodeType == 9) {
			var _stl = [
						QZFL.dom.getScrollTop(this),
						QZFL.dom.getScrollLeft(this)];
		} else {
			var _stl = [this.scrollTop, this.scrollLeft];
		}

		var _st = _stl[0] - top;
		var _sl = _stl[1] - left;

		var n = new QZFL.Tween(this, "_p", QZFL.transitions[ease] || QZFL.transitions.regularEaseOut, 0, 100, 0.5);
		n.onMotionChange = QZFL.event.bind(this, function() {
			var _p = arguments[2], _t = (_stl[0] - _p / 100 * _st), _l = (_stl[1] - _p / 100 * _sl);

			if (this.nodeType == 9) {
				QZFL.dom.setScrollTop(_t, this);
				QZFL.dom.setScrollLeft(_l, this);
			} else {
				this.scrollTop = _t;
				this.scrollLeft = _l;
			}
		});
		n.start();
	};

	QZFL.element.extendFn({
		tween : function(){
			
		},
		
		/**
		 * 娓愬彉鏄剧ず
		 * @param {string} effect 杞崲鏁堟灉,鐩墠鍙敮鎸?resize"
		 * @param {string} ease 鍔ㄧ敾鏁堟灉
		 * @example $e(document).effectShow();
		 */
		
		effectShow : function(effect, ease) {
			this.each(function(el) {
				_easeShowAnimate.apply(el, [true, ease])
			});
			if (effect == "resize") {
				this.each(function(el) {
					_easeAnimate.apply(el, ["Size", null, null, ease])
				});
			}
		},
		/**
		 * 娓愬彉闅愯棌
		 * @param {string} effect 杞崲鏁堟灉,鐩墠鍙敮鎸?resize"
		 * @param {string} ease 鍔ㄧ敾鏁堟灉
		 * @example $e(document).effectHide();
		 */
		effectHide : function(effect, ease) {
			this.each(function(el) {
				_easeShowAnimate.apply(el, [false, ease])
			});
			if (effect == "resize") {
				this.each(function(el) {
					_easeAnimate.apply(el, ["Size", 0, 0, ease])
				});
			}
		},
		/**
		 * 鏀瑰彉灏哄
		 * @param {number} width 瀹藉害
		 * @param {number} height 楂樺害
		 * @param {string} ease 鍔ㄧ敾鏁堟灉
		 * @example $e(document).effectResize(200,200);
		 */
		effectResize : function(width, height, ease) {
			this.each(function(el) {
				_easeAnimate.apply(el, ["Size", width, height, ease])
			});
		},
		/**
		 * 鏀瑰彉浣嶇疆
		 * @param {number} x x鍧愭爣
		 * @param {number} y y鍧愭爣
		 * @param {string} ease 鍔ㄧ敾鏁堟灉
		 * @example $e(document).effectMove(200,200);
		 */
		effectMove : function(x, y, ease) {
			this.each(function(el) {
				_easeAnimate.apply(el, ["XY", x, y, ease])
			});
		},
		/**
		 * 婊氬姩鏉℃粦鍔?		 * @param {number} top 绾靛悜璺濈
		 * @param {number} left 妯悜璺濈
		 * @param {string} ease 鍔ㄧ敾鏁堟灉
		 * @example $e(document).effectScroll(200);
		 */
		effectScroll : function(top, left, ease) {
			this.each(function(el) {
				_easeScroll.apply(el, [top, left, ease])
			});
		}
		// ,
		//
		// effectNotify : function(ease) {
		// this.each(function() {
		// var _c = QZFL.dom.getStyle(this,"backgroundColor");
		// var n = new QZFL.Tween(this, "backgroundColor",
		// QZFL.transitions[ease] || QZFL.transitions.regularEaseOut, "#ffffff",
		// "#ffff00", 0.8);
		//
		// n.onMotionStop = QZFL.event.bind(this,function(){
		// var o = this;
		// setTimeout(function(){
		// var n = new QZFL.Tween(o, "backgroundColor", QZFL.transitions[ease]
		// || QZFL.transitions.regularEaseOut, "#ffff00", "#ffffff", 1);
		// n.onMotionStop = function(){
		// o.style.backgroundColor = "transparent";
		// }
		// n.start();
		// },1000)
		// });
		// n.start();
		// });
		// }
	})
})();



/////////////
//deferred.js
/////////////


/**
 * 瀹炵幇promise妯″紡鐨刣eferred瀵硅薄
 * @param {Function} [func] 寮傛鍑芥暟
 * @param {Array} [args] 鍙傛暟鍒楄〃
 * @return promise瀵硅薄
 * @author hankzhu
 * @version 1.0.0.0
 * @example 瑙乹zfl_proj/trunk/src/widget/examples/deferred
 */
QZFL.Deferred = function(func, args){
    var _slice = Array.prototype.slice
    , Promise = function(){
        this.status = undefined;   //0-''  1-resolve 2-reject
    }
    , Event = {
        _status : {'reject':1, 'resolve':1},
        _init : function(type){ //2绉峵ype
            if(!this.eventList){
                this.eventList = {
                    rejectFuncs : [],
                    resolveFuncs : []
                };
            }
        },
        add : function(type, func){
            this._init(type);
            if(typeof(func) == 'function'){ //杩囨护涓€涓?                if(this.status == type){
                    func.apply(window, this.eventList[type + 'Datas']);
                }else{
                    this.eventList[type + 'Funcs'].push(func);
                }
                this.eventList.added = 1;
            }
            return this;
        },
        trigger : function(type, datas){
            var i, funcs, func;
            if(type in this._status){
                this._init();
                if(this.eventList.added){ //宸茬粡鏈夋敞鍐宒one鎴栬€協ail
                    funcs = this.eventList[type + 'Funcs'];
                    while((func = funcs.shift())){
                        func.apply(window, datas);
                    }
                }else{//鍦ㄨЕ鍙戠殑鏃跺€欙紝杩樻病鏈夌粦瀹氫换浣曚簨浠?                    this.eventList[type + 'Datas'] = datas;
                }
                this.status = type;
            }
        }
    }
    , _promise;
    
    QZFL.object.extend(Promise.prototype, {
        done : function(func){
            return this.add('resolve', func);
        },
        fail : function(func){
            return this.add('reject', func);
        },
        then : function(doneFunc, failFunc){
            return this.add('resolve', doneFunc).add('reject', failFunc);
        },
        resolve : function(){
            this.trigger('resolve', _slice.call(arguments));
        },
        reject : function(){
            this.trigger('reject', _slice.call(arguments));
        },
        state : function(){
            return this.status;
        }
    }, Event);
   
    _promise = new Promise();
    if(!(args instanceof Array)){
        args = [];
    }
    //鍑芥暟浣撳唴閮ㄥ彧鏈変袱涓柟娉?    if(typeof(func) == 'function'){
        //浣滀负鏈€鍚庝竴涓弬鏁?        args.push(_promise);
        func.apply(window, args);
    }
    //鍏ㄩ儴杩斿洖
    return _promise;
};
QZFL.object.extend(QZFL, {
    when : function(promise){
        var _slice = Array.prototype.slice,
            promises = _slice.call(arguments),
            length = promises.length, remain, updateFunc, datas = [], d, s;
            
        remain = (length !== 1 || ( promise && promise.state()) ? length : 0);
        promise = (remain === 1 ? promise : QZFL.Deferred());
        
        //姣忎釜promise瀹屾垚鐨勬椂鍊欓兘浼氭鏌ヨ繖涓?        updateFunc = function(index, data) {
            datas[index] = _slice.call(data);
            if(!(--remain)){
                promise.resolve.apply(promise, datas); //瑙﹀彂done
            }
        };
        if(length > 1){
            for(var i = 0; i < length; i++){
                if(!promises[i].state){ throw new Error('not a promise instance');} //闈瀙romise瀵硅薄
                promises[i].done((function(i){
                    return function(){
                        updateFunc(i, arguments);
                    };
                })(i)).fail(function(){
                    promise.reject(_slice.call(arguments));
                });
            }
        }
        if(!remain) {
            promise.resolve();
        }   
        return promise;   
    }
});



/////////////
//xhr.js
/////////////


/**
 * @fileoverview QZFL XMLHttpRequest缁勪欢澹? * @version 1.1
 * @author ryanzhao, zishunchen, scorpionxu
 */

/**
 * XMLHttpRequest閫氫俊鍣ㄧ粍浠讹紝濡備粖宸茬粡涓嶅お寤鸿浣跨敤锛屾帹鑽愪娇鐢ㄥ熀浜嶫SONP鐨勬暟鎹帴鍙? * 鈥滃皬璺ㄥ煙鈥濇媺鍙栨暟鎹椂锛坅.qq.com涓婄殑椤甸潰鎷夊彇b.qq.com涓婄殑璧勬簮锛夊彲浠ヤ娇鐢紝闇€瑕佸€熷姪/resource/html/xhr_proxy_gbk.html
 * @deprecated 涓嶅お寤鸿浣跨敤鍩轰簬XHR鐨勫墠鍚庡彴閫氫俊鏂瑰紡
 *
 * @class XMLHttpRequest閫氫俊鍣ㄧ粍浠? * @constructor
 * @param {String} actionURI 璇锋眰鍦板潃
 * @param {String} [cname='_xhrInstence_'+QZFL.XHR.counter] 瀵硅薄瀹炰綋鐨勭储寮曞悕锛岄粯璁ゆ槸"_xhrInstence_n"锛宯涓哄簭鍙? * @param {String} [method='POST'] 鍙戦€佹柟寮忥紝闄ら潪鎸囨槑get锛屽惁鍒欏叏閮ㄤ负post
 * @param {Object} [data={}] hashTable褰㈠紡鐨勫瓧鍏? * @param {boolean} [isAsync=true] <strong style="color:red;">Deprecated</strong> 涓嶅湪鎻愪緵鍚屾妯″紡锛屾案杩滀负寮傛
 * @param {boolean} [nocache=false] 鏄惁鏃燾ache
 */
QZFL.XHR = function(actionURL, cname, method, data, isAsync, nocache) {
	var _s = QZFL.XHR,
		prot,
		n;

	cname = cname || ("_xhrInstence_" + _s.counter);

	if (!(_s.instance[cname] instanceof QZFL.XHR)) {
		_s.instance[cname] = this;
		_s.counter++;
	}

	prot = _s.instance[cname]

	prot._name = cname;
	prot._nc = !!nocache;
	prot._method = ((typeof method == 'string' ? method : '').toUpperCase() != "GET") ? "POST" : "GET";
	if(!(prot._uriObj = new QZFL.util.URI(actionURL))){ //URL閮戒笉瑙勮寖灏变笉瑕佺帺浜?		throw (new Error("URL not valid!"));
	}
	prot._uri = actionURL;
	prot._data = data;


	// 瀵瑰鐨勬帴鍙ｇ兢
	/**
	 * 褰撴垚鍔熷洖璋冩椂瑙﹀彂鐨勪簨浠?	 * @param {object} data 鍥炶皟鏁版嵁
	 * @param {object} data.text 鍏跺疄灏辨槸XHR鐨剅esponseText
	 * @param {object} data.xmlDom 鍏跺疄灏辨槸XHR鐨剅esponseXML
	 * @event
	 */
	this.onSuccess = QZFL.emptyFn;

	/**
	 * 褰撻敊璇椂锛岄€氬父鏄綉缁滈棶棰橈紝鎴栬€呭悗鍙版寕鎺?	 * @param {object} msg 閿欒璇村悕瀛?	 * @event
	 */
	this.onError = QZFL.emptyFn;

	/**
	 * 浣跨敤鐨勭紪鐮?	 * @type string
	 */
	this.charset = "gb2312";

	/**
	 * 鍙傛暟鍖杙roxy鐨勮矾寰勶紝涔熷氨鏄湪琚姹傝祫婧愬煙鍚嶄笂鐨勯偅涓法鍩熶唬鐞嗘枃浠秞hr_proxy_gbk.html鐨勪綅缃?	 * @type string
	 */
	this.proxyPath = "";



	return prot;
};

/**
 * @private
 */
QZFL.XHR.instance = {};

/**
 * @private
 */
QZFL.XHR.counter = 0;


/**
 * 鏈綋璺緞
 *
 */
QZFL.XHR.path = location.protocol + '//' + QZFL.config.resourceDomain + "/ac/qzfl/release/expand/xhr_base.js?max_age=864001",



/**
 * 鍙戦€佽姹? *
 * @returns {boolean} 鏄惁鎴愬姛鍙戝嚭
 */
QZFL.XHR.prototype.send = function() {
	var _s = QZFL.XHR,
		fn;
	if (this._method == 'POST' && !this._data) { //can't send POST request with no data
		return false;
	}

	if(typeof this._data == "object"){
		this._data = _s.genHttpParamString(this._data, this.charset);
	}

	if(this._method == 'GET' && this._data){
		this._uri += (this._uri.indexOf("?") < 0 ? "?"  : "&") +  this._data;
	}

	//鍒ゆ柇鏄惁闇€瑕佽法鍩熻姹傛暟鎹?	fn = (location.hostname && (this._uriObj.host != location.hostname)) ? '_DoXsend' : '_DoSend';
	if(_s[fn]){
		return _s[fn](this);
	}else{
		QZFL.imports(_s.path, (function(th){
				return function(){
					_s[fn](th);
				};
			})(this));
		return true;
	}
};


/**
 * @private
 */
QZFL.XHR.genHttpParamString = function(o, cs){
	cs = (cs || "gb2312").toLowerCase();
	var r = [];

	for (var i in o) {
		r.push(i + "=" + ((cs == "utf-8") ? encodeURIComponent(o[i]) : QZFL.string.URIencode(o[i])));
	}

	return r.join("&");
};





/////////////
//cookie.js
/////////////


/**
 * @fileoverview QZFL cookie鏁版嵁澶勭悊
 * @version 1.$Rev: 1921 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */

/**
 * cookie绫?cookie绫诲彲浠ヨ寮€鍙戝緢杞绘澗寰楁帶鍒禼ookie锛屾垜浠彲浠ラ殢鎰忓鍔犱慨鏀瑰拰鍒犻櫎cookie锛屼篃鍙互杞绘槗璁剧疆cookie鐨刾ath, domain, expire绛変俊鎭? *
 * @namespace QZFL.cookie
 */
QZFL.cookie = {
	/**
	 * 璁剧疆涓€涓猚ookie,杩樻湁涓€鐐归渶瑕佹敞鎰忕殑锛屽湪qq.com涓嬫槸鏃犳硶鑾峰彇qzone.qq.com鐨刢ookie锛屽弽姝zone.qq.com涓嬭兘鑾峰彇鍒皅q.com鐨勬墍鏈塩ookie.
	 * 绠€鍗曞緱璇达紝瀛愬煙鍙互鑾峰彇鏍瑰煙涓嬬殑cookie, 浣嗘槸鏍瑰煙鏃犳硶鑾峰彇瀛愬煙涓嬬殑cookie.
	 * @param {String} name cookie鍚嶇О
	 * @param {String} value cookie鍊?	 * @param {String} domain 鎵€鍦ㄥ煙鍚?	 * @param {String} path 鎵€鍦ㄨ矾寰?	 * @param {Number} hour 瀛樻椿鏃堕棿锛屽崟浣?灏忔椂
	 * @return {Boolean} 鏄惁鎴愬姛
	 * @example
	 *  QZFL.cookie.set('value1',QZFL.dom.get('t1').value,"qzone.qq.com","/v5",24); //璁剧疆cookie
	 */
	set : function(name, value, domain, path, hour) {
		if (hour) {
			var expire = new Date();
			expire.setTime(expire.getTime() + 3600000 * hour);
		}
		document.cookie = name + "=" + value + "; " + (hour ? ("expires=" + expire.toGMTString() + "; ") : "") + (path ? ("path=" + path + "; ") : "path=/; ") + (domain ? ("domain=" + domain + ";") : ("domain=" + QZFL.config.domainPrefix + ";"));
		return true;
	},

	/**
	 * 鑾峰彇鎸囧畾鍚嶇О鐨刢ookie鍊?	 *
	 * @param {String} name cookie鍚嶇О
	 * @return {String} 鑾峰彇鍒扮殑cookie鍊?	 * @example
	 * 		QZFL.cookie.get('value1'); //鑾峰彇cookie
	 */
	get : function(name) {
		//ryan
		//var s = ' ' + document.cookie + ';', pos;
		//return (pos = s.indexOf(' ' + name + '=')) > -1 ? s.slice(pos += name.length + 2, s.indexOf(';', pos)) : '';
		
		var r = new RegExp("(?:^|;+|\\s+)" + name + "=([^;]*)"), m = document.cookie.match(r);
		return (!m ? "" : m[1]);
	},

	/**
	 * 鍒犻櫎鎸囧畾cookie,澶嶅啓涓鸿繃鏈?	 *
	 * @param {String} name cookie鍚嶇О
	 * @param {String} domain 鎵€鍦ㄥ煙
	 * @param {String} path 鎵€鍦ㄨ矾寰?	 * @example
	 * 		QZFL.cookie.del('value1'); //鍒犻櫎cookie
	 */
	del : function(name, domain, path) {
		document.cookie = name + "=; expires=Mon, 26 Jul 1997 05:00:00 GMT; " + (path ? ("path=" + path + "; ") : "path=/; ") + (domain ? ("domain=" + domain + ";") : ("domain=" + QZFL.config.domainPrefix + ";"));
	}
};




/////////////
//debug.js
/////////////


/**
 * @fileoverview 鐢ㄤ簬璋冭瘯绌洪棿鐨勯敊璇俊鎭? * @version 1.$Rev: 1921 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */

/**
 * 閿欒璋冭瘯绫? *
 * @namespace QZFL.debug
 */
QZFL.debug = {
	/**
	 * 閿欒瀵硅薄
	 */
	errorLogs : [],

	/**
	 * 鍚敤Qzone璋冭瘯妯″紡锛岄敊璇細璁板綍鍒板璞′腑
	 */
	startDebug : function() {
		/**
		 * 涓虹獥鍙ｅ姞鍏ラ敊璇鐞?		 *
		 * @ignore
		 * @param {Object} e
		 */
		window.onerror = function(msg,url,line) {
			var urls = (url || "").replace(/\\/g,"/").split("/");
			QZFL.console.print(msg + "<br/>" + urls[urls.length - 1] + " (line:" + line + ")",1);
			QZFL.debug.errorLogs.push(msg);
			return false;
		}
	},

	/**
	 * 鍋滄Qzone璋冭瘯妯″紡
	 */
	stopDebug : function() {
		/**
		 * 涓虹獥鍙ｅ姞鍏ラ敊璇鐞?		 *
		 * @ignore
		 */
		window.onerror = null;
	},

	/**
	 * 娓呴櫎鎵€鏈夐敊璇俊鎭?	 */
	clearErrorLog : function() {
		this.errorLogs = [];
	},

	showLog : function() {
		var o = ENV.get("debug_out");
		if (!!o) {
			o.innerHTML = QZFL.string.nl2br(QZFL.string.escHTML(this.errorLogs.join("\n")));
		}
	},

	getLogString : function() {
		return (this.errorLogs.join("\n"));
	}
};

/**
 * runtime澶勭悊宸ュ叿闈欐€佺被
 *
 * @namespace runtime澶勭悊宸ュ叿闈欐€佺被
 */
QZFL.runTime = (function() {
	/**
	 * 鏄惁debug鐜
	 *
	 * @return {Boolean} 鏄惁鍛?	 */
	function isDebugMode() {
		return (QZFL.config.debugLevel > 1);
	}

	/**
	 * log璁板綍鍣?	 *
	 * @ignore
	 * @param {String} msg 淇℃伅璁板綍鍣?	 */
	function log(msg, type) {
		var info;
		if (isDebugMode()) {
			info = msg + '\n=STACK=\n' + stack();
		} else {
			if (type == 'error') {
				info = msg;
			} else if (type == 'warn') {
				// TBD
			}
		}
		QZFL.debug.errorLogs.push(info);
	}

	/**
	 * 璀﹀憡淇℃伅璁板綍
	 *
	 * @param {String} sf 淇℃伅妯″紡
	 * @param {All} args 濉厖鍙傛暟
	 */
	function warn(sf, args) {
		log(QZFL.string.write.apply(QZFL.string, arguments), 'warn');
	}

	/**
	 * 閿欒淇℃伅璁板綍
	 *
	 * @param {String} sf 淇℃伅妯″紡
	 * @param {All} args 濉厖鍙傛暟
	 */
	function error(sf, args) {
		log(QZFL.string.write.apply(QZFL.string, arguments), 'error');
	}

	/**
	 * 鑾峰彇褰撳墠鐨勮繍琛屽爢鏍堜俊鎭?	 *
	 * @param {Error} e 鍙€夛紝褰撴椂鐨勫紓甯稿璞?	 * @param {Arguments} a 鍙€夛紝褰撴椂鐨勫弬鏁拌〃
	 * @return {String} 鍫嗘爤淇℃伅
	 */
	function stack(e, a) {
		function genTrace(ee, aa) {
			if (ee.stack) {
				return ee.stack;
			} else if (ee.message.indexOf("\nBacktrace:\n") >= 0) {
				var cnt = 0;
				return ee.message.split("\nBacktrace:\n")[1].replace(/\s*\n\s*/g, function() {
					cnt++;
					return (cnt % 2 == 0) ? "\n" : " @ ";
				});
			} else {
				var entry = (aa.callee == stack) ? aa.callee.caller : aa.callee;
				var eas = entry.arguments;
				var r = [];
				for (var i = 0, len = eas.length; i < len; i++) {
					r.push((typeof eas[i] == 'undefined') ? ("<u>") : ((eas[i] === null) ? ("<n>") : (eas[i])));
				}
				var fnp = /function\s+([^\s\(]+)\(/;
				var fname = fnp.test(entry.toString()) ? (fnp.exec(entry.toString())[1]) : ("<ANON>");
				return (fname + "(" + r.join() + ");").replace(/\n/g, "");
			}
		}

		var res;

		if ((e instanceof Error) && (typeof arguments == 'object') && (!!arguments.callee)) {
			res = genTrace(e, a);
		} else {
			try {
				({}).sds();
			} catch (err) {
				res = genTrace(err, arguments);
			}
		}

		return res.replace(/\n/g, " <= ");
	}

	return {
		/**
		 * 鑾峰彇褰撳墠鐨勮繍琛屽爢鏍堜俊鎭?		 *
		 * @param {Error} e 鍙€夛紝褰撴椂鐨勫紓甯稿璞?		 * @param {Arguments} a 鍙€夛紝褰撴椂鐨勫弬鏁拌〃
		 * @return {String} 鍫嗘爤淇℃伅
		 */
		stack : stack,
		/**
		 * 璀﹀憡淇℃伅璁板綍
		 *
		 * @param {String} sf 淇℃伅妯″紡
		 * @param {All} args 濉厖鍙傛暟
		 */
		warn : warn,
		/**
		 * 閿欒淇℃伅璁板綍
		 *
		 * @param {String} sf 淇℃伅妯″紡
		 * @param {All} args 濉厖鍙傛暟
		 */
		error : error,

		/**
		 * 鏄惁璋冭瘯妯″紡
		 */
		isDebugMode : isDebugMode
	};

})();



/////////////
//js_loader.js
/////////////


/**
 * @fileoverview QZFL Javascript Loader
 * @version 1.$Rev: 1918 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 17:35:04 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */

/**
 * Js Loader锛宩s鑴氭湰寮傛鍔犺浇
 *
 * @constructor
 * @example
 * 		var t=new QZFL.JsLoader();
 *		t.onload = function(){};
 *		t.load("/qzone/v5/tips_diamond.js", null, {"charset":"utf-8"});
 */
QZFL.JsLoader = function() {
	/**
	 * 褰搄s涓嬭浇瀹屾垚鏃?	 *
	 * @event
	 */
	this.onload = QZFL.emptyFn;

	/**
	 * 缃戠粶闂涓嬭浇鏈畬鎴愭椂
	 *
	 * @event
	 */
	this.onerror = QZFL.emptyFn;
};


/**
 * 鍔ㄦ€佸姞杞絁S
 *
 * @param {string} src javascript鏂囦欢鍦板潃
 * @param {Object} doc document
 * @param {string} [opt] 褰撲负瀛楃涓叉椂锛屾寚瀹歝harset
 * @param {object} [opt] 褰撲负瀵硅薄琛ㄦ椂锛屾寚瀹?script>鏍囩鐨勫悇绉嶅睘鎬? *
 */
QZFL.JsLoader.prototype.load = function(src, doc, opt){
	var opts = {}, t = typeof(opt), o = this;

	if (t == "string") {
		opts.charset = opt;
	} else if (t == "object") {
		opts = opt;
	}

	opts.charset = opts.charset || "gb2312";

	//TO DO  涓€涓槻閲嶅姞杞戒紭鍖?	QZFL.userAgent.ie ? setTimeout(function(){
		o._load(src, doc || document, opts);
	}, 0) : o._load(src, doc || document, opts);
};

QZFL.JsLoader.count = 0;
QZFL.JsLoader._idleInstancesIDQueue = [];

/**
 * 寮傛鍔犺浇js鑴氭湰
 *
 * @param {Object} sId
 * @param {Object} src
 * @param {Object} doc
 * @param {Object} opts
 * @ignore
 * @private
 */
QZFL.JsLoader.prototype._load = (function() {
	var _ie = QZFL.userAgent.ie,
		_doc = document,
		idp = QZFL.JsLoader._idleInstancesIDQueue,
		_rm = QZFL.dom.removeElement,
		_ae = QZFL.event.addEvent,
		docMode = _doc.documentMode;
	return function(/*sId, */src, doc, opts){
		var o = this,
			tmp,
			k,
			head = doc.head || doc.getElementsByTagName("head")[0] || doc.body,
			_new = false,
			_js;

		if(!(_js = doc.getElementById(idp.pop())) || (QZFL.userAgent.ie && QZFL.userAgent.ie > 8)){
			_js = doc.createElement("script");
			_js.id = "_qz_jsloader_" + (++QZFL.JsLoader.count);
			_new = true;
		}

		// 澶勭悊鍔犺浇鎴愬姛鐨勫洖璋?		_ae(_js, (_ie && _ie < 10 ? "readystatechange" : "load"), function(){
			//ie鐨勫鐞嗛€昏緫
			if (!_js || _ie && _ie < 10 && ((typeof docMode == 'undefined' || docMode < 10) ? (_js.readyState != 'loaded') : (_js.readyState != 'complete'))) {
				return;
			}
			_ie && idp.push(_js.id);

			o.onload();
			!_ie && _rm(_js);
			_js = o = null;
		});

		if (!_ie) {
			_ae(_js, 'error', function(){
				_ie && idp.push(_js.id);

				o.onerror();
				!_ie && _rm(_js);
				_js = o = null;
			})
		}

		for (k in opts) {
			if (typeof(tmp = opts[k]) == "string" && k.toLowerCase() != "src") {
				_js.setAttribute(k, tmp);
			}
		}

		_new && head.appendChild(_js);

		_js.src = src;

		opts = null;
	};
}) ();

/**
 * JsLoader鐨勭畝鍐?閬垮厤琚垎鏋愬嚭鏉? * @deprecated 涓嶅缓璁娇鐢?鍙仛鍏煎
 * @ignore
 */
QZFL["js"+"Loader"]=QZFL.JsLoader;




/////////////
//imports.js
/////////////


/**
 * @fileoverview QZFL Javascript Imports锛屾敮鎸佸苟琛屽姞杞姐€佹牴鎹畁amespace鍔犺浇
 * @version 1.$Rev: 1544 $
 * @author QzoneWebGroup, ($LastChangedBy: joltwang $)
 */
/**
 * 寮傛鍔犺浇涓€浜涜剼鏈簱 by ryan
 * @param {Object} sources
 * @param {Object} succCallback
 * @param {Object} opts
 */
QZFL.imports = function(sources, succCallback, opts){
	var errCallback, url, len, countId, counter, scb, ecb, i, isFn = QZFL.lang.isFunction;
	opts = QZFL.lang.isString(opts) ? {
		charset: opts
	} : (opts || {});
	opts.charset = opts.charset || 'utf-8';
	var errCallback = isFn(opts.errCallback) ? opts.errCallback : QZFL.emptyFn;
	succCallback = isFn(succCallback) ? succCallback : QZFL.emptyFn;
	
	if (typeof(sources) == "string") {
		url = QZFL.imports.getUrl(sources);
		QZFL.imports.load(url, succCallback, errCallback, opts);
	} else if (QZFL.lang.isArray(sources)) {
		countId = QZFL.imports.getCountId();
		len = QZFL.imports.counters[countId] = sources.length;
		counter = 0;
		scb = function(){
			counter++;
			if (counter == len) {
				if (isFn(succCallback)) succCallback();
			}
			delete QZFL.imports.counters[countId];
		};
		ecb = function(){
			if (isFn(errCallback)) errCallback();
			QZFL.imports.counters[countId];
		};
		
		for (i = 0; i < len; i++) {
			url = QZFL.imports.getUrl(sources[i]);
			QZFL.imports.load(url, scb, ecb, opts);
		}
	}
};

QZFL.imports.getUrl = function(url){
	return QZFL.string.isURL(url) ?
		url
			:
		(QZFL.imports._indirectUrlRE.test(url) ?
			url
				:
			(QZFL.config.staticServer + url + '.js'));
};

QZFL.imports.urlCache = {};
QZFL.imports.counters = {};
QZFL.imports.count = 0;
QZFL.imports._indirectUrlRE = /^(?:\.{1,2})?\//;

QZFL.imports.getCountId = function(){
	return 'imports' + QZFL.imports.count++;
};

QZFL.imports.load = function(url, scb, ecb, opt){
	if (QZFL.imports.urlCache[url] === true) {
		setTimeout(function(){
			if (QZFL.lang.isFunction(scb)) scb()
		}, 0);
		return;
	}
	if (!QZFL.imports.urlCache[url]) {
		QZFL.imports.urlCache[url] = [];
		var loader = new QZFL.JsLoader();
		loader.onload = function(){
			QZFL.imports.execFnQueue(QZFL.imports.urlCache[url], 1);
			QZFL.imports.urlCache[url] = true;
		};
		loader.onerror = function(){
			QZFL.imports.execFnQueue(QZFL.imports.urlCache[url], 0);
			QZFL.imports.urlCache[url] = null;
			delete QZFL.imports.urlCache[url];
		};
		loader.load(url, null, opt);
	}
	QZFL.imports.urlCache[url].push([ecb, scb]);
};

QZFL.imports.execFnQueue = function(arFn, isSuccess){
	var f;
	while (arFn.length) {
		f = arFn.shift()[isSuccess];
		if (QZFL.lang.isFunction(f)) {
			setTimeout((function(fn){
				return fn
			})(f), 0);
		}
	}
};




/////////////
//form_sender.js
/////////////


/**
 * @fileoverview QZFL Form Submit Class
 * @version 1.$Rev: 1897 $
 * @author QzoneWebGroup, ($LastChangedBy: scorpionxu $)
 * @lastUpdate $Date: 2010-12-27 20:59:34 +0800 (鍛ㄤ竴, 27 鍗佷簩鏈?2010) $
 */

/**
 * FormSender閫氫俊鍣ㄧ被,寤鸿鍐欐搷浣滀娇鐢? *
 * @param {String} actionURL 璇锋眰鍦板潃
 * @param {String} [method] 鍙戦€佹柟寮忥紝闄ら潪鎸囨槑get锛屽惁鍒欏叏閮ㄤ负post
 * @param {Object} [data] hashTable褰㈠紡鐨勫瓧鍏? * @param {String} [charset="gb2312"] 浜庡悗鍙版暟鎹氦浜掔殑瀛楃闆? * @constructor
 * @namespace QZFL.FormSender
 *
 * cgi杩斿洖妯℃澘: <html><head><meta http-equiv="Content-Type" content="text/html;
 * charset=gb2312" /></head> <body><script type="text/javascript">
 * document.domain="qq.com"; frameElement.callback({JSON:"Data"}); </script></body></html>
 * @example
 * 		var fs = new QZFL.FormSender(APPLY_ENTRY_RIGHT,"post",{"hUin": getParameter("uin"),"vUin":checkLogin(),"msg":$("msg-area").value, "rd": Math.random()}, "utf-8");
 *		fs.onSuccess = function(re) {};
 *		fs.onError = function() {};
 *		fs.send();
 *
 */
QZFL.FormSender = function(actionURL, method, data, charset) {

	/**
	 * form鐨勫悕绉帮紝榛樿涓?_fpInstence_ + 璁℃暟
	 *
	 * @type string
	 */
	this.name = "_fpInstence_" + QZFL.FormSender.counter;
	QZFL.FormSender.instance[this.name] = this;
	QZFL.FormSender.counter++;

	if(typeof(actionURL) == 'object' && actionURL.nodeType == 1 && actionURL.tagName == 'FORM'){
		this.instanceForm = actionURL;
	}else{ //standard mode

		/**
		 * 鏁版嵁鍙戦€佹柟寮?		 *
		 * @type string
		 */
		this.method = method || "POST";

		/**
		 * 鏁版嵁璇锋眰鍦板潃
		 *
		 * @type string
		 */
		this.uri = actionURL;

		/**
		 * 鏁版嵁鍙傛暟琛?		 *
		 * @type object
		 */
		this.data = (typeof(data) == "object" || typeof(data) == 'string') ? data : null;
		this.proxyURL = (typeof(charset) == 'string' && charset.toUpperCase() == "UTF-8")
				? QZFL.config.FSHelperPage.replace(/_gbk/, "_utf8")
				: QZFL.config.FSHelperPage;

	}


	this._sender = null;


	/**
	 * 鏈嶅姟鍣ㄦ纭搷搴旀椂鐨勫鐞?	 *
	 * @event
	 */
	this.onSuccess = QZFL.emptyFn;

	/**
	 * 鏈嶅姟鍣ㄦ棤鍝嶅簲鎴栭瀹氱殑涓嶆甯稿搷搴斿鐞?	 *
	 * @event
	 */
	this.onError = QZFL.emptyFn;
};

QZFL.FormSender.instance = {};
QZFL.FormSender.counter = 0;

QZFL.FormSender._errCodeMap = {
	999 : {
		msg : 'Connection or Server error'
	}
};



QZFL.FormSender.pluginsPool = {
	"formHandler" : []
	, "onErrorHandler" : []
};

QZFL.FormSender._pluginsRunner = function(pType, data){
	var _s = QZFL.FormSender,
		l = _s.pluginsPool[pType],
		t = data,
		len;

	if(l && (len = l.length)){
		for(var i = 0; i < len; ++i){
			if(typeof(l[i]) == "function"){
				t = l[i](t);
			}
		}
	}

	return t;
};

/**
 * 鍙戦€佽姹? *
 * @return {Boolean} 鏄惁鎴愬姛
 */
QZFL.FormSender.prototype.send = function() {
	this.startTime = +new Date;
	if (this.method == 'POST' && this.data == null) {
		return false;
	}

	function clear(o) {
		o._sender = o._sender.callback = o._sender.errorCallback = o._sender.onreadystatechange = null;
		if (QZFL.userAgent.safari || QZFL.userAgent.opera) {
			setTimeout('QZFL.dom.removeElement(document.getElementById("_fp_frm_' + o.name + '"))', 50);
		} else {
			QZFL.dom.removeElement(document.getElementById("_fp_frm_" + o.name));
		}
		// 瀹屾垚浜嗕竴娆¤姹?		o.endTime = +new Date;
		QZFL.FormSender._pluginsRunner('onRequestComplete', o);
		o.instanceForm = null;
	}

	if (this._sender === null || this._sender === void(0)) {
		var sender = this.instanceForm ? QZFL.dom.createNamedElement("iframe", "_fp_frm_" + this.name) : document.createElement("iframe");
		sender.id = "_fp_frm_" + this.name;
		sender.style.cssText = "width:0;height:0;border-width:0;display:none;";
		document.body.appendChild(sender);

		sender.callback = QZFL.event.bind(this, function(o) {
					clearTimeout(timer);
					this.resultArgs = arguments;
					this.msg = 'ok';
					this.onSuccess(o);
					clear(this);
				});
		sender.errorCallback = QZFL.event.bind(this, function(o) {
					clearTimeout(timer);
					this.resultArgs = arguments;
					this.msg = QZFL.FormSender._errCodeMap[999].msg;
					this.onError(o);

					QZFL.FormSender._pluginsRunner('onErrorHandler', this);
					clear(this);
				});

		if (typeof(sender.onreadystatechange) != 'undefined') {
			sender.onreadystatechange = QZFL.event.bind(this, function() {
				if (this._sender.readyState == 'complete' && this._sender.submited) {
					clear(this);
					this.onError(QZFL.FormSender._errCodeMap[999]);

					QZFL.FormSender._pluginsRunner('onErrorHandler', this);

				}
			});
		} else {
			var timer = setTimeout(QZFL.event.bind(this, function() {
					try {
						var _t = this._sender.contentWindow.location.href;
						if (_t.indexOf(this.uri) == 0) {
							clearTimeout(timer);
							clear(this);
							this.onError(QZFL.FormSender._errCodeMap[999]);

							QZFL.FormSender._pluginsRunner('onErrorHandler', this);

						}
					} catch (err) {
						clearTimeout(timer);
						clear(this);
						this.onError(QZFL.FormSender._errCodeMap[999]);

						QZFL.FormSender._pluginsRunner('onErrorHandler', this);

					}
				}), 1500);
		}

		this._sender = sender;
	}

	if(!this.instanceForm){
		this.proxyURL.replace('http:','');
		this._sender.src = this.proxyURL;
	}else{
		this.instanceForm.target = (sender.name = sender.id);
		this._sender.submited = true;
		this.instanceForm.submit();
	}
	return true;
};

/**
 * QZFL.FormSender瀵硅薄鑷瘉鏂规硶锛岀敤娉?ins=ins.destroy();
 *
 * @return {Object} null鐢ㄦ潵澶嶅啓寮曠敤鏈韩
 */
QZFL.FormSender.prototype.destroy = function() {
	var n = this.name;
	delete QZFL.FormSender.instance[n]._sender;
	QZFL.FormSender.instance[n]._sender = null;
	delete QZFL.FormSender.instance[n];
	QZFL.FormSender.counter--;
	return null;
};











/////////////
//json.js
/////////////


/**
 * @fileoverview QZFL JSON绫? * @version 1.$Rev: 1895 $
 * @author QzoneWebGroup, ($LastChangedBy: scorpionxu $)
 * @lastUpdate $Date: 2010-12-27 20:19:39 +0800 (鍛ㄤ竴, 27 鍗佷簩鏈?2010) $
 */

/**
 * JSONGetter閫氫俊鍣ㄧ被,寤鸿浣跨敤杩涜璇绘搷浣滅殑鏃跺€欎娇鐢?
 * @class JSONGetter閫氫俊鍣ㄧ被
 * @param {string} actionURI 璇锋眰鍦板潃
 * @param {string} cname 鍙€夛紝瀵硅薄瀹炰綋鐨勭储寮曞悕锛岄粯璁ゆ槸"_jsonInstence_n"锛宯涓哄簭鍙? * @param {object} data 鍙€夛紝hashTable褰㈠紡鐨勫瓧鍏? * @param {string} charset 鎵€鎷夊彇鏁版嵁鐨勫瓧绗﹂泦
 * @param {boolean} [junctionMode=false] 浣跨敤鎻掑叆script鏍囩鐨勬柟寮忔媺鍙? * @constructor
 * @example
 * 		var _loader = new QZFL.JSONGetter(GET_QUESTIONS_URL, void (0), {"uin": getParameter("uin"), "rd": Math.random()}, "utf-8");
 *		_loader.onSuccess = function(re){};
 *		_loader.send("_Callback");
 *		_loader.onError = function(){};
 */
QZFL.JSONGetter = function(actionURL, cname, data, charset, junctionMode) {
	if (QZFL.object.getType(cname) != "string") {
		cname = "_jsonInstence_" + (QZFL.JSONGetter.counter + 1);
	}
	
	var prot = QZFL.JSONGetter.instance[cname];
	if (prot instanceof QZFL.JSONGetter) {
		//ignore
	} else {
		QZFL.JSONGetter.instance[cname] = prot = this;
		QZFL.JSONGetter.counter++;

		prot._name = cname;
		prot._sender = null;
		prot._timer = null;

		// 璁板綍寮€濮嬭姹傜殑鏃堕棿鐐?		this.startTime = +new Date;
		
		/**
		 * 鍥炶皟鎴愬姛鎵ц
		 * 
		 * @event
		 */
		this.onSuccess = QZFL.emptyFn;

		/**
		 * 瑙ｉ噴澶辫触
		 * 
		 * @event
		 */
		this.onError = QZFL.emptyFn;
		
		/**
		 * 褰撴暟鎹秴鏃剁殑鏃跺€?		 * 
		 * @event
		 */
		this.onTimeout = QZFL.emptyFn;
		
		/**
		 * 瓒呮椂璁剧疆,榛樿5绉掗挓
		 */
		this.timeout = 5000;
		
		/**
		 * 鎶涘嚭娓呯悊鎺ュ彛
		 */
		this.clear = QZFL.emptyFn;

		this._baseClear = function(){
			this._waiting = false;
			this._squeue = [];
			this._equeue = [];
			this.onSuccess = this.onError = QZFL.emptyFn;
			this.clear = null;
		};
	}

	prot._uri = actionURL;
	prot._data = (data && (QZFL.object.getType(data) == "object" || QZFL.object.getType(data) == "string")) ? data : null;
	prot._charset = (QZFL.object.getType(charset) != 'string') ? QZFL.config.defaultDataCharacterSet : charset;
	prot._jMode = !!junctionMode;

	return prot;
};

QZFL.JSONGetter.instance = {};
QZFL.JSONGetter.counter = 0;

QZFL.JSONGetter._errCodeMap = {
	999 : {
		msg : 'Connection or Server error.'
	},
	998 : {
		msg : 'Connection to Server timeout.'
	}
};

QZFL.JSONGetter.genHttpParamString = function(o){
	var r = [];

	for (var i in o) {
		r.push(i + "=" + encodeURIComponent(o[i]));
	}

	return r.join("&");
};

/**
 * 娣诲姞涓€涓垚鍔熷洖璋冨嚱鏁? * @param {Function} f
 */
QZFL.JSONGetter.prototype.addOnSuccess = function(f){
	if(typeof(f) == "function"){
		if(this._squeue && this._squeue.push){

		}else{
			this._squeue = [];
		}
		this._squeue.push(f);
	}
};


QZFL.JSONGetter._runFnQueue = function(q, resultArgs, th){
	var f;
	if(q && q.length){
		while(q.length > 0){
			f = q.shift();
			if(typeof(f) == "function"){
				f.apply(th ? th : null, resultArgs);
			}
		}
	}
	th.endTime = +new Date;
	th.resultArgs = resultArgs;
	QZFL.JSONGetter._pluginsRunner("onRequestComplete", th); //sds 鐢ㄦ彃浠舵潵璺戜竴涓媢rl鍋氭彃鎺ュ姛鑳斤紝濡傚弽CSRF缁勪欢
};

/**
 * 娣诲姞涓€涓け璐ュ洖璋冨嚱鏁? * @param {Function} f
 */
QZFL.JSONGetter.prototype.addOnError = function(f){
	if(typeof(f) == "function"){
		if(this._equeue && this._equeue.push){

		}else{
			this._equeue = [];
		}
		this._equeue.push(f);
	}
};


QZFL.JSONGetter.pluginsPool = {
	"srcStringHandler" : []
	, "onErrorHandler" : []
	, "onRequestComplete" : []
};

QZFL.JSONGetter._pluginsRunner = function(pType, data){
	var _s = QZFL.JSONGetter,
		l = _s.pluginsPool[pType],
		t = data,
		len;

	if(l && (len = l.length)){
		for(var i = 0; i < len; ++i){
			if(typeof(l[i]) == "function"){
				t = l[i](t);
			}
		}
	}

	return t;
};


QZFL.JSONGetter.prototype.send = function(callbackFnName) {
	if(this._waiting){ //宸茬粡鍦ㄨ姹備腑閭ｄ箞灏变笉鍐嶅彂璇锋眰浜?		return;
	}

	var clear,
		cfn = (QZFL.object.getType(callbackFnName) != 'string') ? "callback" : callbackFnName,
		da = this._uri;
		
	if(this._data){
		da += (da.indexOf("?") < 0 ? "?" : "&") + ((typeof(this._data) == "object") ? QZFL.JSONGetter.genHttpParamString(this._data) : this._data);
	}

	da = QZFL.JSONGetter._pluginsRunner("srcStringHandler", da); //sds 鐢ㄦ彃浠舵潵璺戜竴涓媢rl鍋氭彃鎺ュ姛鑳斤紝濡傚弽CSRF缁勪欢
	
	//浼犺涓殑jMode... 娆茬煡璇︽儏锛岃鍜ㄨ鍝撳摀鍚屽
	if(this._jMode){
		window[cfn] = this.onSuccess;
		var _sd = new QZFL.JsLoader();
		_sd.onerror = this.onError;
		_sd.load(da,void(0),this._charset);
		return;
	}

	//璁剧疆瓒呮椂鐐?	this._timer = setTimeout(
			(function(th){
				return function(){
						//QZFL.console.print("jsonGetter timeout", 3);
						//TODO timeout can't push in success or failed... zishunchen 
						th._waiting = false;
						th.onTimeout();
					};
				})(this),
			this.timeout
		);
	
	//IE10 Customer Preview 杩欓噷涓嶈兘鐢╠f鐢氳嚦hf浜嗭紝鎼炲埌涓嬮潰鐨勫垎鏀幓
	if (QZFL.userAgent.ie && (typeof document.documentMode == 'undefined' || document.documentMode < 10) && !(QZFL.userAgent.beta && navigator.appVersion.indexOf("Trident\/4.0") > -1)) { // IE8涔嬪墠鐨勬柟妗?纭畾瑕佸钩绋宠縼绉讳箞
		var df = document.createDocumentFragment()
			,sender = document.createElement("script")
			;//sds 鍔犱釜IE > 9鍏煎
		
		sender.charset = this._charset;
		
		this._senderDoc = df;
		this._sender = sender;
		
		//鍥炶皟鍚庢竻鐞?		this.clear = clear = function(o){
			clearTimeout(o._timer);
			if (o._sender) {
				o._sender.onreadystatechange = null;
			}
			df['callback'] = df['_Callback'] = df[cfn] = null;
			df = o._senderDoc = o._sender = null;
			o._baseClear();
		};
		
		//鎴愬姛鍥炶皟
		df['callback'] = df['_Callback'] = df[cfn] = (function(th){
				return (function(){
					th._waiting = false;
					th.onSuccess.apply(th, arguments);
					QZFL.JSONGetter._runFnQueue(th._squeue, arguments, th);
					clear(th);
				});
			})(this);
		
		//鐢ㄦ潵妯℃嫙ie鍦ㄥ姞杞藉け璐ョ殑鎯呭喌
		if(QZFL.userAgent.ie<9){
			sender.onreadystatechange = (function(th){
				return (function(){
					if (th._sender && th._sender.readyState == "loaded") {
						try {
							th._onError();
						} catch (ignore) {}
					}
				});
			})(this);
		} else {
			sender.onerror = (function(th){
				return (function(){
					try {
						th._onError();
					} catch (ignore) {}
				});
			})(this);
		}
			
		this._waiting = true;
		
		df.appendChild(sender);
		this._sender.src = da;

	} else {
		//鍥炶皟鍚庢竻鐞?		this.clear = clear = function(o) {
			//QZFL.console.print(o._timer);
			clearTimeout(o._timer);
			o._baseClear();
		};
		
		//鍏ㄥ眬鎵ц鐨勫嚱鏁扮敵鏄?		window[cfn] = function() {
			QZFL.JSONGetter.args = arguments;
		};	
		
		// 鍏ㄥ眬鎵ц瀹屾瘯鍥炴敹鑺傜偣瑙﹀彂鍥炶皟
		var callback = (function(th) {
			return function() {
				th.onSuccess.apply(th, QZFL.JSONGetter.args);
				QZFL.JSONGetter._runFnQueue(th._squeue, QZFL.JSONGetter.args, th);
				QZFL.JSONGetter.args = [];
				clear(th);
			}
		})(this);
		// 澶辫触鍥炶皟
		var _ecb = (function(th){
			return (function() {
				th._waiting = false;
				var _eo = QZFL.JSONGetter._errCodeMap[999];
				th.msg = _eo.msg;
				th.onError(_eo);
				QZFL.JSONGetter._runFnQueue(th._equeue, [_eo], th);
				clear(th);
			});
		})(this);

		var h = document.getElementsByTagName('head'), node;
		h = h && h[0] || document.body;
		if (!h)
			return;
		var baseElement = h.getElementsByTagName('base')[0];
		node = document.createElement('script');
		node.charset = this._charset || 'utf-8';
		node.onload = function () {
			this.onload = null;
			if (node.parentNode) {
				h.removeChild(node);
			}
			callback();
			node = void(0);
		};
		node.onerror = function () {
			this.onerror = null;
			_ecb();
		}
		node.src = da;
		node.src.replace('http:','');
		baseElement ? h.insertBefore(node, baseElement) : h.appendChild(node);
	}
};
QZFL.JSONGetter.prototype._onError = function() {
	this._waiting = false;
	var _eo = QZFL.JSONGetter._errCodeMap[999];
	this.msg = _eo.msg;
	this.onError(_eo);
	QZFL.JSONGetter._runFnQueue(this._equeue, [_eo], this);

	QZFL.JSONGetter._pluginsRunner("onErrorHandler", this); //sds 鐢ㄦ彃浠舵潵璺戜竴涓媢rl鍋氭彃鎺ュ姛鑳斤紝濡傚弽CSRF缁勪欢


	this.clear(this);
};

/**
 * QZFL.JSONGetter瀵硅薄鑷瘉鏂规硶锛岀敤娉?ins=ins.destroy();
 * @deprecated 杩欎釜娌℃湁瀛樺湪鐨勪环鍊? * @returns {object} null鐢ㄦ潵澶嶅啓寮曠敤鏈韩
 */
QZFL.JSONGetter.prototype.destroy = QZFL.emptyFn;




















/***************************************** :) *****************************************************/
/**
 * 鐩墠宸茬粡鑳藉畬鎴愭暟鎹姹傚姛鑳斤紝瑕佹敞鎰忥紝鐜板湪鐨勮皟鐢ㄦ柟寮忓拰浠ュ墠鐨勪笉鍚? * 鏈畬鎴愬姛鑳斤細
 * 	1. 瀵筽roxy cache閲宒f鎴栬€協rame杩涜寤舵椂娓呯悊锛屼繚鎸佸湪涓€涓垨鑰呬袱涓氨澶熶簡
 * 	2. 涓婃姤鍔熻兘
 * 	3. addOnSuccess, addOnError
 * 	4. 瀵笽E8 beta鐗堣繘琛屾彁绀猴紝涓嶅啀鏀寔锛岀幇鍦ㄦ槸鐢╢rame瀹炵幇鐨? * /

QZFL.JSONGetterBeta = function(url){
	this.url = url;
	this.charset = QZFL.config.defaultDataCharacterSet;
	this.onTimeout = this.onSuccess = this.onError = QZFL.emptyFn;
};
QZFL.JSONGetterBeta.prototype.setCharset = function(charset){
	if (typeof(charset) == 'string') {
		this.charset = charset;
	}
};
QZFL.JSONGetterBeta.prototype.setQueryString = function(data){
	var type;
	if (data && ((type = typeof(data)) == 'object' || type == 'string')) {
		if (type == 'object') {
			var r = [];
			for (var k in data) {
				r.push(k + "=" + encodeURIComponent(data[k]));
			}
			data = r.join("&");
		}
		this.url += (this.url.indexOf("?") < 0 ? "?" : "&") + data;
	}
};
QZFL.JSONGetterBeta.prototype.send = function(cbFnName){
	cbFnName = cbFnName || 'callback';
	var me = this, proxy = QZFL.JSONGetterBeta.getProxy(), tmp;
	if (QZFL.JSONGetterBeta.isUseDF) {
		var scrpt = proxy.createElement("script");
		scrpt.charset = this.charset;
		proxy.appendChild(scrpt);
		
		proxy[cbFnName] = function(){
			proxy.requesting = false;
			me.onSuccess.apply(null, Array.prototype.slice.call(arguments));
			scrpt.removeNode(true);
			QZFL.console.print('request finish : ' + me.url);
			scrpt = scrpt.onreadystatechange = me = proxy = proxy[cbFnName] = null;
		};
		
		scrpt.onreadystatechange = function(){
			if (scrpt.readyState == "loaded") {
				proxy.requesting = false;
				me.onError({
					ret: 999,
					msg: 'Connection or Server error.'
				});
				scrpt.removeNode(true);
				QZFL.console.print('request Error : ' + me.url);
				scrpt = scrpt.onreadystatechange = me = proxy = proxy[cbFnName] = null;
			}
		};
		
		proxy.requesting = true;
		scrpt.src = this.url;
	} else {
		proxy.style.width = proxy.style.height = proxy.style.borderWidth = "0";
		
		proxy.callback = function(){
			proxy.requesting = false;
			me.onSuccess.apply(null, Array.prototype.slice.call(arguments));
			var win = proxy.contentWindow;
			clearTimeout(win.timer);
			var scrpts = win.document.getElementsByTagName('script');
			for (var i = 0, l = scrpts.length; i < l; i++) {
				QZFL.dom.removeElement(scrpts[i]);
			}
			QZFL.console.print('request finish : ' + me.url);
			me = proxy = proxy.callback = proxy.errorCallback = null;
		};
		proxy.errorCallback = function(){
			proxy.requesting = false;
			me.onError.apply(null, [{
				ret: 999,
				msg: 'Connection or Server error.'
			}]);
			var win = proxy.contentWindow;
			clearTimeout(win.timer);
			var scrpts = win.document.getElementsByTagName('script');
			for (var i = 0, l = scrpts.length; i < l; i++) {
				QZFL.dom.removeElement(scrpts[i]);
			}
			QZFL.console.print('request Error : ' + me.url);
			me = proxy = proxy.callback = proxy.errorCallback = null;
		};
		var dm = (document.domain == location.host) ? '' : 'document.domain="' + document.domain + '";', 
			html = '<html><head><meta http-equiv="Content-type" content="text/html; charset=' + this.charset + '"/></head><body><script>' + dm + ';function ' + cbFnName + '(){frameElement.callback.apply(null, arguments);}<\/script><script charset="' + this.charset + '" src="' + this.url + '"><\/script><script>timer=setTimeout(frameElement.errorCallback,50);<\/script></body></html>';
		
		proxy.requesting = true;
		if (QZFL.userAgent.opera || QZFL.userAgent.firefox < 3) {
			proxy.src = "javascript:'" + html + "'";
			document.body.appendChild(proxy);
		} else {
			document.body.appendChild(proxy);
			(tmp = proxy.contentWindow.document).open('text/html');
			tmp.write(html);
			tmp.close();
		}
	}
};
QZFL.JSONGetterBeta.getProxy = function(){
	for (var p, i = 0, len = QZFL.JSONGetterBeta.proxy.length; i < len; i++) {
		if ((p = QZFL.JSONGetterBeta.proxy[i]) && !p.requesting) {
			QZFL.console.print('鎵惧埌绗? + i + '涓唬鐞嗗彲鐢?);
			return p;
		}
	}
	QZFL.console.print('娌℃湁鍙敤鐨勪唬鐞嗭紝鍒涘缓涓€涓柊鐨?);
	QZFL.JSONGetterBeta.proxy.push(p = QZFL.JSONGetterBeta.isUseDF ? document.createDocumentFragment() : document.createElement("iframe"));
	return p;
};
QZFL.JSONGetterBeta.proxy = [];
QZFL.JSONGetterBeta.isUseDF = QZFL.userAgent.ie && !QZFL.userAgent.beta;


//浠ヤ笅鏄嚑涓祴璇曠敤渚?var jg = new QZFL.JSONGetterBeta('http://u.qzone.qq.com/cgi-bin/qzone_static_widget?fs=1&uin=20050606&timestamp=0');
jg.onSuccess = function(o){QZFL.console.print(o['_2_0']._uname_);};
jg.send('staticData_Callback');

var jg2 = new QZFL.JSONGetterBeta('http://g.qzone.qq.com/fcg-bin/cgi_emotion_list.fcg?uin=20050606&loginUin=0&s=820043');
jg2.onSuccess = function(o){QZFL.console.print(o.visitcount);};
jg2.send('visitCountCallBack');

var jg1 = new QZFL.JSONGetterBeta('http://n.qzone.qq.com/cgi-bin/pvuv/set_pvuv?uin=20050606&r=0.39620088664296915');
jg1.onSuccess = function(o){QZFL.console.print(o.todayPV);};
jg1.send('QZonePGVDataCallBack1');

var jg3 = new QZFL.JSONGetterBeta('http://u.qzone.qq.com/cgi-bin/qzone_static_widget?fs=1&uin=20050606&timestamp=0&r=' + Math.random());
jg3.onSuccess = function(o){QZFL.console.print(o['_2_0']._uname_);};
jg3.send('staticData_Callback');

var jg4 = new QZFL.JSONGetterBeta('http://n.qzone.qq.com/cgi-bin/pvuv/set_pvuv?uin=20050606&r=0.39620088664296915&r=' + Math.random());
jg4.onSuccess = function(o){QZFL.console.print(o.todayPV);};
jg4.send('QZonePGVDataCallBack1');

var jg5 = new QZFL.JSONGetterBeta('http://g.qzone.qq.com/fcg-bin/cgi_emotion_list.fcg?uin=20050606&loginUin=0&s=820043&r=' + Math.random());
jg5.onSuccess = function(o){QZFL.console.print(o.visitcount);};
jg5.send('visitCountCallBack');

 * 
 * 
 */



/////////////
//ping_sender.js
/////////////


/*****************************************************************************************/

/**
 * @fileoverview 鍙戜竴涓畝鐭璯et璇锋眰鐨勭粍浠? * @author scorpionxu
 * @version 1.0
 */


window.QZFL = window.QZFL || {};

/**
 * 绠€鍗昰et璇锋眰鍙戦€佸櫒
 * @namespace
 * @param {string} url 璇锋眰url
 * @param {number} [t = 0] 璇锋眰寤惰繜鏃跺欢锛屽崟浣峬s
 * @param {object} [opts = {}] 鍙€夊弬鏁板寘
 * @param {function} [opts.load = QZFL.emptyFn] 鎴愬姛鍥炶皟
 *     鍥炶皟妯″瀷
 *          param {object} info 鍥炶皟淇℃伅璇存槑
 *          param {number} info.duration 鎬诲欢杩熸椂闂? *          param {string} info.type 鍥炶皟绫诲瀷 'load', 'error', 'timeout' 鎴愬姛锛岄敊璇紝瓒呮椂 涓夌被鍥炶皟
 *         function(info){
 *         }
 * @param {function} [opts.error = QZFL.emptyFn] 澶辫触鍥炶皟
 * @param {function} [opts.timeout = QZFL.emptyFn] 瓒呮椂鍥炶皟
 * @param {number} [opts.timeoutValue = opts.timeout ? 5000 : undefined] 瓒呮椂鏃堕棿
 *
 */
QZFL.pingSender = function(url, t, opts){
	var _s = QZFL.pingSender,
		iid,
		img;

	if(!url){
		return;
	}

	opts = opts || {};
	
	iid = "sndImg_" + _s._sndCount++;
	img = _s._sndPool[iid] = new Image();
	img.iid = iid;
	img.onload = img.onerror = img.ontimeout = (function(t){
		return function(evt){
			evt = evt || window.event || { type : 'timeout' };
			void(typeof(opts[evt.type]) == 'function' ? setTimeout(
				(function(et, ti){
					return function(){
						opts[et]({ 'type' : et, 'duration' : ((new Date()).getTime() - ti) });
					};
				})(evt.type, t._s_)
				, 0) : 0);
			QZFL.pingSender._clearFn(evt, t);
		};
	})(img);

	(typeof(opts.timeout) == 'function') && setTimeout(function(){
					img.ontimeout && img.ontimeout({ type : 'timeout' });
				}, (typeof(opts.timeoutValue) == 'number' ? Math.max(100, opts.timeoutValue) : 5000));

	void((typeof(t) == 'number') ? setTimeout(function(){
		img._s_ = (new Date()).getTime();
		img.src = url;
	}, (t = Math.max(0, t))) : (img.src = url));
};

/**
 *
 *
 * @private
 *
 *
 */
QZFL.pingSender._sndPool = {};

/**
 *
 *
 * @private
 *
 *
 */
QZFL.pingSender._sndCount = 0;

/**
 *
 *
 * @private
 *
 *
 */
QZFL.pingSender._clearFn = function(evt, ref){
	//evt = evt || window.event;
	var _s = QZFL.pingSender;
	if(ref){
		_s._sndPool[ref.iid] = ref.onload = ref.onerror = ref.ontimeout = ref._s_ = null;
		delete _s._sndPool[ref.iid];
		_s._sndCount--;
		ref = null;
	}
};




/////////////
//media.js
/////////////


/**
 * @fileoverview QZFL 澶氬獟浣撶被
 * @version 1.$Rev: 1924 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 18:52:16 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */
/**
 * 澧炲己瀵筬lash, wmp绛夊濯掍綋鎺т欢鐨勫鐞? *
 * @namespace QZFL.media
 */
QZFL.media = {
	_tempImageList : [],

	_flashVersion : null,
	/*
	 * 鑾峰彇鍥剧墖淇℃伅锛屽闀跨煭杈规瘮渚嬶紝浠ュ強鍝潯鏄暱杈瑰摢鏉℃槸鐭竟
	 * img Element||String 寰呭垎鏋愬浘鐗囩殑鑺傜偣鎴栬€卻rc
	 * opts{
	 *		ow:鍘熷瀹藉害
	 *		oh:鍘熷楂樺害
	 *		errCallback
	 * }
	 * return {
	 *		direction : [闀胯竟闀垮害灞炴€у悕,鐭竟闀垮害灞炴€у悕],
	 *		rate : 闀胯竟鍜岀煭杈圭殑姣斿€?	 }
	 */
	getImageInfo : (function(){
		var _getInfo = function(img,callback,opts){
			if(img){
				var _w = opts.ow || img.width, _h = opts.oh || img.height, r,ls,ss,d;
				if(_w && _h){
					if(_w >=_h){
						ls = _w;
						ss = _h;
						d = ["width","height"];
					}else{
						ls = _h;
						ss = _w;
						d = ["height","width"];
					}
					r = {
						direction:d,
						rate : ls/ ss,
						longSize :ls,
						shortSize :ss
					};
					r.ow = _w;
					r.oh = _h;
				}
				QZFL.lang.isFunction(callback) && callback(img,r,opts);
			}
		};

		return function( callback, opts){
			opts = opts || {};
			if( QZFL.lang.isString( opts.trueSrc ) ){
				var _i = new Image();
				_i.onload = (function(ele, cb, p){
					return function(){
						_getInfo( ele, cb, p );
						ele = ele.onerror =  ele.onload = null;
					};
				})(_i, callback, opts);

				_i.onerror =  (function(ele, cb, p){
					return function(){
						if (typeof(p.errCallback) == 'function') {
							p.errCallback();
						}
						ele = ele.onerror =  ele.onload = null;
					};
				})(_i, callback, opts);

				_i.src = opts.trueSrc;
			}else if( QZFL.lang.isElement( opts.img ) ){
				_getInfo( opts.img, callback, opts );
			}
		};
	})(),
	/**
	 * 鎸夋瘮渚嬭皟鑺傚浘鐨勫ぇ灏?	 * @example
	 * 			<img src="b.gif" onload="QZFL.media.adjustImageSize(200,150,'http://www.true.com/true.jpg')" />
	 * @param {Number} w 鏈熸湜瀹藉害涓婇檺
	 * @param {Number} h 鏈熸湜楂樺害涓婇檺
	 * @param {String} trueSrc 鐪熸鍦板浘鐗囨簮
	 * @param {Function} callback 鍥剧墖澶у皬璋冩暣瀹屾垚鍚庣殑鍥炶皟
	 */
	adjustImageSize : function(w, h, trueSrc, callback,errCallback) {
		var opts = {trueSrc:trueSrc,callback:function(cb){
			return function(o, type, ew, eh, p){
				//鍏煎涔呮柟娉曪紝涓嶅緱涓嶄互杩欎釜褰㈠紡浼犲弬缁欏洖璋冿紝浣嗗鍔犱竴涓弬鏁帮紝鎶婄収鐗囧帇缂╃殑淇℃伅鍏ㄩ儴浼犺繘鍘?				QZFL.lang.isFunction(cb) && cb(o, ew, eh, null, p.ow, p.oh,p);
			};
		}(callback),errCallback:errCallback};
		QZFL.media.reduceImage( 0, w, h, opts);
	},

	/**
	 * 鐢熸垚flash鐨勬弿杩癏TML
	 *
	 * @param {Object} flashArguments 浠ashTable鎻忚堪鐨刦lash鍙傛暟闆嗗悎,flashUrl璇风敤"src"
	 * @param {QZFL.media.SWFVersion} requiredVersion
	 *            鎵€闇€瑕佺殑flashPlayer鐨勭増鏈紝QZFL.media.SWFVersion鐨勫疄渚?	 * @param {String} flashPlayerCID flash鍦↖E涓娇鐢ㄧ殑classID,鍙€?	 * @return {String} 鐢熸垚鐨凥TML浠ｇ爜
	 * @example
	 * 			var swf_html = QZFL.media.getFlashHtml({
	 *									"src" :"your flash url",
	 *									"width" : "100%",
	 *									"height" : "100%",
	 *									"allowScriptAccess" : "always",
	 *									"id" : "avatar",
	 *									"name" : "avatar",
	 *									"wmode" : "opaque",
	 *                                  "noSrc" : false
	 *						});
	 */
	getFlashHtml : function(flashArguments, requiredVersion, flashPlayerCID) {
		var _attrs = [],
			_params = [];

		for (var k in flashArguments) {
			switch (k) {
				case "noSrc" :
				case "movie" :
					continue; //sds 杩欓噷鏄笉澶勭悊鐨勭壒鎬?					break;
				case "id" :
				case "name" :
				case "width" :
				case "height" :
				case "style" :
					if(typeof(flashArguments[k]) != 'undefined'){
						_attrs.push(' ', k, '="', flashArguments[k], '"');
					}
					break;
				case "src" :
					if (QZFL.userAgent.ie) {
						_params.push('<param name="movie" value="', (flashArguments.noSrc ? "" : flashArguments[k]), '"/>');
					//	_params.push('<param name="movie" value="', flashArguments[k], '"/>');
					}else{
						_attrs.push(' data="', (flashArguments.noSrc ? "" : flashArguments[k]), '"');
					}
					break;
				default :
					_params.push('<param name="', k, '" value="', flashArguments[k], '" />');
			}
		}
		
		
		if (QZFL.userAgent.ie) {
            if(QZFL.userAgent.ie > 10){
                _attrs.push(' classid="clsid:', flashPlayerCID || 'D27CDB6E-AE6D-11cf-96B8-444553540000', '"',
                    ' data="',flashArguments.src||'','"');
            }else{
                _attrs.push(' classid="clsid:', flashPlayerCID || 'D27CDB6E-AE6D-11cf-96B8-444553540000', '"');
            }
		}else{
			_attrs.push(' type="application/x-shockwave-flash"');
		}
	 	
		if (requiredVersion && (requiredVersion instanceof QZFL.media.SWFVersion)) {
			var _ver = QZFL.media.getFlashVersion().major,
				_needVer = requiredVersion.major;

			//褰撴病鏈夊畨瑁呭苟涓斿簲鐢ㄦ病鏈夊埢鎰忔寚瀹氱殑鏃跺€欙紝璧癈odebase璺嚎
			_attrs.push(' codeBase="//fpdownload.macromedia.com/get/flashplayer/current/swflash.cab#version=', requiredVersion, '"');
		}

		return "<object" + _attrs.join("") + ">" + _params.join("") + "</object>";
	},
	

	/**
	 * 瀵硅皟鐢ㄦ柟缁欏嚭鐨勪竴涓妭鐐规彃鍏ラ渶瑕佺殑flash object
	 * @param {HTMLElement} containerElement 瀹瑰櫒鑺傜偣锛屽繀椤?	 * @param {Object} flashArguments @see QZFL.media.getFlashHtml()
	 * @return {boolean} 鏄惁鎴愬姛
	 *
	 *
	 */
	insertFlash : function(containerElement, flashArguments){
		if(!containerElement || typeof(containerElement.innerHTML) == "undefined"){
			return false;
		}
		flashArguments = flashArguments || {};
		flashArguments.src = flashArguments.src || "";
		flashArguments.width = flashArguments.width || "100%";
		flashArguments.height = flashArguments.height || "100%";

		flashArguments.noSrc = true;

		containerElement.innerHTML = QZFL.media.getFlashHtml(flashArguments);
		var f = containerElement.firstChild;

		if(QZFL.userAgent.ie){
			setTimeout(function(){
					try{ f.LoadMovie(0, flashArguments.src); }catch(ign){}
				}, 0);
		}else{
			f.setAttribute("data", flashArguments.src);
		}
		return true;
	},

	/*
	 * 鐢熸垚Windows Media Player鐨凥TML鎻忚堪
	 * @example
	 * 			var wmphtml=QZFL.media.getWMMHtml({id:'qzfl_media',name:'qzfl_wmp',width:'300px',height:'200px',src:'#',style:''});
	 */
	getWMMHtml : function(wmpArguments, cid) {
		var params = [],
			objArgm = [];

		for (var k in wmpArguments) {
			switch (k) {
				case "id" :
				case "width" :
				case "height" :
				case "style" :
				case "src" :
					objArgm.push(' ', k, '="', wmpArguments[k], '"');
					break;
				default :
					objArgm.push(' ', k, '="', wmpArguments[k], '"');
					params.push('<param name="', k, '" value="', wmpArguments[k], '" />');
			}
		}
		if (wmpArguments["src"]) {
			params.push('<param name="URL" value="', wmpArguments["src"], '" />');
		}

		if (QZFL.userAgent.ie) {
			return '<object classid="' + (cid || "clsid:6BF52A52-394A-11D3-B153-00C04F79FAA6") + '" ' + objArgm.join("") + '>' + params.join("") + '</object>';
		} else {
			return '<embed ' + objArgm.join("") + '></embed>';
		}
	}
};

/**
 * flash鐗堟湰鍙锋樉绀哄櫒
 *
 * @param {Number} arguments...
 *            鍙彉鍙傛暟锛屼竴绯诲垪鏁板瓧锛屽墠4涓槸flash鐨勫洓娈电増鏈彿锛屼篃鍙互鍗曟暟缁勪紶鍏ワ紝涔熷彲浠ュ彧浣跨敤涓€涓暣鏁拌〃绀轰富鐗堟湰鍙? * @constructor
 */
QZFL.media.SWFVersion = function() {
	var a;
	if (arguments.length > 1) {
		a = arg2arr(arguments);
	} else if (arguments.length == 1) {
		if (typeof(arguments[0]) == "object") {
			a = arguments[0];
		} else if (typeof arguments[0] == 'number') {
			a = [arguments[0]];
		} else {
			a = [];
		}
	} else {
		a = [];
	}

	this.major = parseInt(a[0], 10) || 0;
	this.minor = parseInt(a[1], 10) || 0;
	this.rev = parseInt(a[2], 10) || 0;
	this.add = parseInt(a[3], 10) || 0;
};

/**
 * flash鐗堟湰鏄剧ず鍣ㄥ簭鍒楀寲鏂规硶
 *
 * @param {Object} spliter 鐗堟湰鍙锋樉绀猴紝鏁板瓧鍒嗛殧绗? * @return {String} 涓€涓弿杩癴lashPlayer鐗堟湰鍙风殑瀛楃涓? */
QZFL.media.SWFVersion.prototype.toString = function(spliter) {
	return ([this.major, this.minor, this.rev, this.add])
			.join((typeof spliter == 'undefined') ? "," : spliter);
};
/**
 * flash鐗堟湰鏄剧ず鍣ㄥ簭鍒楀寲鏂规硶
 *
 * @return {String} 涓€涓弿杩癴lashPlayer鐗堟湰鍙风殑鏁板瓧
 */
QZFL.media.SWFVersion.prototype.toNumber = function() {
	var se = 0.001;
	return this.major + this.minor * se + this.rev * se * se + this.add * se * se * se;
};

/**
 * 鑾峰彇褰撳墠娴忚鍣ㄤ笂瀹夎鐨刦lashPlayer鐨勭増鏈紝鏈畨瑁呰繑鍥炵殑瀹炰緥toNumber()鏂规硶绛変簬0
 *
 * @return {Object} 杩斿洖QZFL.media.SWFVersion鐨勫疄渚?{major,minor,rev,add}
 * @example
 * 			QZFL.media.getFlashVersion();
 */
QZFL.media.getFlashVersion = function() {
	if (!QZFL.media._flashVersion) {
		var resv = 0;
		if (navigator.plugins && navigator.mimeTypes.length) {
			var x = navigator.plugins['Shockwave Flash'];
			if (x && x.description) {
				resv = x.description.replace(/(?:[a-z]|[A-Z]|\s)+/, "")
						.replace(/(?:\s+r|\s+b[0-9]+)/, ".").split(".");
			}
		} else {
			try {
				for (var i = (resv = 6), axo = new Object(); axo != null; ++i) {
					axo = new ActiveXObject("ShockwaveFlash.ShockwaveFlash." + i);
					resv = i;
				}
			} catch (e) {
				if (resv == 6) {
					resv = 0;
				}//6閮芥病鏈夊氨褰撴病瀹夎鍚?				resv = Math.max(resv - 1, 0);
			}
			try {
				resv = new QZFL.media.SWFVersion(axo.GetVariable("$version")
						.split(" ")[1].split(","));
			} catch (ignore) {}
		}
		if (!(resv instanceof QZFL.media.SWFVersion)) {
			resv = new QZFL.media.SWFVersion(resv);
		}

		if (resv.major < 3) {
			resv.major = 0;
		}
		QZFL.media._flashVersion = resv;
	}
	return QZFL.media._flashVersion;
};
/*
 * @author heliosliu
 * @param {number} type	鍘嬬缉绫诲瀷,1鏄寜鐭竟鍘嬬缉锛?鏄寜闀胯竟鍘嬬缉
 * @param {number} ew	鏈€澶у搴? * @param {number} eh	鏈€澶ч珮搴? * @param {object} opts{
 *		direction : [闀胯竟闀垮害灞炴€у悕,鐭竟闀垮害灞炴€у悕],
 *		rate : 闀胯竟鍜岀煭杈圭殑姣斿€?
 *		callback : 鍥炶皟鍑芥暟,
 *		ow:	鍘熷瀹藉害
 *		oh:	鍘熷楂樺害
 *		img	寰呭帇缂╃殑鍥剧墖
 *		trueSrc : 瀹為檯鍦板潃
 *  }
 * @example <img src="b.gif" onload="QZFL.media.reduceImage(1,60,45,{callback:QZFL.emptyFn,eroCallback:QZFL.emptyFn,trueSrc:'http://www.true.com/true.jpg'});" /> 
 */
QZFL.media.reduceImage = (function(){
	var doReduce = function(o, type, ew, eh, p, cb){
		var rl, k;
		if(p.rate==1){
			p.direction[0] = ( ew>eh ? 'height' : 'width');
			p.direction[1] = ( ew>eh ? 'width' : 'height');
		}
		rl = ( p.direction[type] == "width" ? ew : eh );
		type ?
			( ( ( rl>p.shortSize ) ? ( rl = p.shortSize ) : 1 ) && ( p.k = p.shortSize/rl ) )
		:
			( ( ( rl>p.longSize ) ? ( rl = p.longSize ) : 1 ) && ( p.k = p.longSize/rl ) );
		o.setAttribute(p.direction[type],rl);

		QZFL.lang.isFunction(cb) && cb(o, type, ew, eh, p);
	};
	return function( type, ew, eh, opts){//img, 
		opts = opts || {};
		opts.img = ( QZFL.lang.isNode( opts.img ) ? opts.img : QZFL.event.getTarget() );
		opts.img.onload=null;
		opts.trueSrc && (opts.img.src = opts.trueSrc);
		if(opts.img){
			if( ! ( opts.direction && opts.rate && opts.longSize && opts.shortSize ) ){
				r = QZFL.media.getImageInfo(function(o,p){
					if(!o||!p){
						return;
					}
					doReduce(opts.img, type, ew, eh, p, opts.callback)
				},opts);
			}else{
				doReduce(opts.img, type, ew, eh, opts, opts.callback)
			}
		}
	};
})();
QZFL.media.imagePlusUrl = '//' + QZFL.config.resourceDomain + '/ac/qzfl/release/widget/smart_image.js?max_age=1209603';
/*
 * @info 璇︾粏瑙乻mart_image.js
 */
QZFL.media.smartImage = function(w, h, params){
	params = params||{};
	params.img = ( QZFL.lang.isNode( params.img ) ? params.img : QZFL.event.getTarget() );
	QZFL.imports(QZFL.media.imagePlusUrl, (function(w, h ,params){
		return function(){
			QZFL.media.smartImage(w, h,params);
		};
	})(w, h ,params));
};
/*
 * @info 鏍规嵁缁欏畾鐨勫浘鐗囨瘮渚嬬殑涓嶅悓鍋氬嚭涓嶅悓绛栫暐鐨勫帇缂? * @info 璇︾粏瑙乻mart_image.js
 */
QZFL.media.reduceImgByRule = function(  ew, eh, opts, cb){
	opts = opts || {} ;
	opts.img = ( QZFL.lang.isNode( opts.img ) ? opts.img : QZFL.event.getTarget() );
	QZFL.imports(QZFL.media.imagePlusUrl, (function( ew, eh, opts, cb){
		return function(){
			QZFL.media.reduceImgByRule( ew, eh, opts, cb);
		};
	})( ew, eh, opts, cb));
};
//sds 浼间箮涔熷彲浠ュ共鎺変簡
//edit by youyeelu
/*
QZFL.media._changeFlashSrc = function(src, installVer, needVer) {
	//褰撶敤鎴峰畨瑁呬簡flash player锛屼絾鏄病鏈夎揪鍒板簲鐢ㄨ姹傜殑鐗堟湰鐨勬椂鍊欙紝璧板揩閫熷畨瑁呰矾绾?	if( installVer >= 6 && needVer > installVer){
		src = "http://qzs.qq.com/qzone/flashinstall.swf";
	}
	return src;
};
*/



/////////////
//shareobject.js
/////////////


/**
 * @fileoverview QZFL ShareObject 瀛樺偍绫? * @version 1.$Rev: 1921 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */

/**
 * Qzone 鐢ㄤ簬鍩烘湰瀛樺偍瀹㈡埛绔熀鏈俊鎭? *
 * @namespace QZFL.shareObject
 */
QZFL.shareObject = {};

/**
 * 鍒濆鍖杝hareObject
 *
 * @param {String} [path] 瑕佹媺璧风殑shareObject swf鍦板潃
 * @example
 * 			QZFL.shareObject.create();
 */
QZFL.shareObject.create = function(path) {
	if (typeof(path) == 'undefined') {
		path = QZFL.config.defaultShareObject;
	}
	var t = new QZFL.shareObject.DataBase(path);
};

QZFL.shareObject.instance = {};
QZFL.shareObject.refCount = 0;

/**
 * 鑾峰彇涓€涓彲鐢ㄧ殑SO
 *
 * @return {Object} ShareObject瀹炰緥
 * @example
 * 			var so=QZFL.shareObject.getValidSo();//鑾峰彇涓€涓彲鐢ㄧ殑ShareObject
 */
QZFL.shareObject.getValidSO = function() {
	var cnt = QZFL.shareObject.refCount + 1;
	for (var i = 1; i < cnt + 1; ++i) {
		if (QZFL.shareObject.instance["so_" + i] && QZFL.shareObject.instance["so_" + i]._ready) {
			return QZFL.shareObject.instance["so_" + i];
		}
	}
	return null;
};

/**
 * 鑾峰彇鏁版嵁鐨勯潤鎬佹柟娉? * @param {String} s 閿悕
 * @return {All} 瀛樺偍鐨勫€? * @example
 * 			QZFL.shareObject.get(key);//璇诲彇
 */
QZFL.shareObject.get = function(s){
	var o = QZFL.shareObject.getValidSO();
	if(o) return o.get(s);
	else return void(0);
};



/**
 * 瀛樺叆鏁版嵁鐨勯潤鎬佹柟娉? * @param {String} k 閿悕
 * @param {All} v 鍊? * @return {Boolean} 鏄惁鎴愬姛
 * @example
 * 			QZFL.shareObject.set(key,value);//瀛樺叆
 */
QZFL.shareObject.set =	function(k, v){
	var o = QZFL.shareObject.getValidSO();
	if(o) return o.set(k, v);
	else return false;
};


/**
 * flash瀹㈡埛绔瓨鍌ㄥ櫒鏋勯€犲櫒锛屽彧浼氬湪椤甸潰鍒涘缓涓€涓疄渚? *
 * @constructor
 */
QZFL.shareObject.DataBase = function(soUrl) {
	/* 闄愬埗鏁扮洰 */
	if (QZFL.shareObject.refCount > 0) {
		return QZFL.shareObject.instance["so_1"];
	}

	//TODO 鏆傛椂鍘绘帀瀵筍O鐨勫垽鏂?	//var fv = QZFL.media.getFlashVersion();
	//if (fv.toNumber() < 8) {
	//	rt.error("flash player version is too low to build a shareObject!");
	//	return null;
	//}

	this._ready = false;

	QZFL.shareObject.refCount++;
	var c = document.createElement("div");
	c.style.height = "0px";
	c.style.overflow = "hidden";//add by joltwang 褰卞搷v6鑷畾涔夌毊鑲?	//c.className = "qz_v6_so_container"
	//c.style.marginTop = "-1px"; //removed by scorpionxu 杩欎釜鏈変弗閲嶉棶棰?	document.body.appendChild(c);
	c.innerHTML = QZFL.media.getFlashHtml({
		src : soUrl,
		id : "__so" + QZFL.shareObject.refCount,
		width : 1, //缁欎竴鐐瑰彲瑙佹€?		height : 0,
		allowscriptaccess : "always"
	});
	this.ele = $("__so" + QZFL.shareObject.refCount);

	QZFL.shareObject.instance["so_" + QZFL.shareObject.refCount] = this;
};

/**
 * 浠ユ寚瀹氶敭鍚嶏紝瀛樺偍涓€涓瓧绗︿覆鎻忚堪鐨勬暟鎹? *
 * @param {String} key 瀛樺偍鐨勯敭鍚? * @param {String} value 瀛樺偍鐨勫€硷紝蹇呴』鏄瓧绗︿覆绫诲瀷
 * @return {Boolean} 鏄惁鎴愬姛
 */
QZFL.shareObject.DataBase.prototype.set = function(key, value) {
	if (this._ready) {
		this.ele.set("seed", Math.random());
		this.ele.set(key, value);
		this.ele.flush();
		return true;
	} else {
		return false;
	}
};
/**
 * 鍒犻櫎涓€涓凡缁忓瓨鍌ㄧ殑閿? *
 * @param {String} key 瀛樺偍鐨勯敭鍚? * @return {Boolean} 鏄惁鎴愬姛
 */
QZFL.shareObject.DataBase.prototype.del = function(key) {
	if (this._ready) {
		this.ele.set("seed", Math.random());
		this.ele.set(key, void(0));
		this.ele.flush();
		return true;
	} else {
		return false;
	}
};
/**
 * 鑾峰彇鎸囧畾閿悕鐨勬暟鎹? *
 * @param {String} key 鎸囧畾鐨勯敭鍚? * @return {String/Object} 寰楀埌鐨勫€硷紝null琛ㄧず涓嶅瓨鍦? */
QZFL.shareObject.DataBase.prototype.get = function(key) {
	return (this._ready) ? (this.ele.get(key)) : null;
};
/**
 * 娓呴櫎鎵€鏈夋暟鎹紝鎱庣敤锛? *
 * @return {Boolean} 鏄惁鎴愬姛
 */
QZFL.shareObject.DataBase.prototype.clear = function() {
	if (this._ready) {
		this.ele.clear();
		return true;
	} else {
		return false;
	}
};
/**
 * 鑾峰彇鏁版嵁闀垮害
 *
 * @return {Number} -1琛ㄧず澶辫触
 */
QZFL.shareObject.DataBase.prototype.getDataSize = function() {
	if (this._ready) {
		return this.ele.getSize();
	} else {
		return -1;
	}
};
/**
 * 鍙戣捣杩炴帴
 *
 * @param {String} url
 * @param {String} succFnName
 * @param {String} errFnName
 * @param {Object} data
 * @return {Boolean} 鏄惁鎴愬姛
 */
QZFL.shareObject.DataBase.prototype.load = function(url, succFnName,
		errFnName, data) {
	if (this._ready) {
		this.ele.load(url, succFnName, errFnName, data);
		return true;
	} else {
		return false;
	}
};
/**
 * @ignore
 */
QZFL.shareObject.DataBase.prototype.setReady = function() {
	this._ready = true;
};

/**
 * Flash鍒濆鍖栨柟娉? *
 * @return {String} flash鍐呴儴浣跨敤
 */
function getShareObjectPrefix() {
	QZFL.shareObject.instance["so_" + QZFL.shareObject.refCount].setReady();
	// return location.host.match(/\w+/)[0];
	return location.hostname.replace(".qzone.qq.com","");
}

/**
 * 澶嶅埗鍒板壀璐存澘
 *
 * @param {String} value 瑕佸鍒剁殑鍊? * @return {Boolean} 鏄惁鎴愬姛
 */
QZFL.shareObject.DataBase.prototype.setClipboard = function(value) {
	if (this._ready && isString(value)) {
		this.ele.setClipboard(value);
		return true;
	} else {
		return false;
	}
};



/////////////
//dragdrop_shell.js
/////////////


/**
 * @fileoverview QZFL 鎷栨嫿绫? * @version 1.$Rev: 1723 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2010-04-08 19:26:57 +0800 (鍛ㄥ洓, 08 鍥涙湀 2010) $
 */

/**
 * 鎷栨嫿绠＄悊鍣紝璐熻矗瀵筪om瀵硅薄杩涜鎷栨嫿鐨勭粦瀹氥€? *
 * @namespace QZFL.dragdrop
 */
QZFL.dragdrop = {

	path : location.protocol + '//' + QZFL.config.resourceDomain + "/ac/qzfl/release/widget/dragdrop.js?max_age=864001",

	/**
	 * 鎷栨嫿姹狅紝鐢ㄦ潵璁板綍宸茬粡娉ㄥ唽鎷栨嫿鐨勫璞?	 */
	dragdropPool : {},

	/**
	 * 鎷栨嫿瀵硅薄涓存椂ID.
	 *
	 * @ignore
	 */
	count : 0,


	/**
	 * 娉ㄥ唽鎷栨嫿瀵硅薄, 娉ㄥ唽鍚庯紝杩斿洖鎷栨斁鎻忚堪瀵硅薄
	 *
	 * @param {HTMLElement} handle 鎺ㄦ嫿鐨勫璞＄殑handler
	 * @param {HTMLElement} target 闇€瑕佹帹鎷界殑瀵硅薄
	 * @param {Object} opts 鍙傛暟 {range,rangeElement,x,y,ghost,ghostStyle} <br/><br/>
	 *            range [top,left,bottom,right]
	 *            鎸囧畾涓€涓皝闂殑鎷栨斁鍖哄煙,鍙傛暟鍙互涓嶆瘮鍏ㄨ缃暀绌烘垨璁剧疆涓洪潪鏁板瓧濡俷ull[top,left,bottom,right]
	 *            鏄痭umber<br/> rangeElement [element,[top,left,bottom,right],isStatic]
	 *            鍒跺畾鎷栨斁鍖哄煙鐨勫璞★紝闄愬埗鐗╀綋鍙兘鍦ㄨ繖涓尯鍩熷唴鎷栨斁銆?[top,left,bottom,right]
	 *            鏄痓oolean,rangeElement鍜宼arget蹇呴』鏄悓涓€涓潗鏍囩郴锛岃€屼笖target蹇呴』鍦╮angeElement鍐?	 *            銆俰sStatic {boolean} 鏄寚 rangeElement 娌℃湁浣跨敤鐙珛鐨勫潗鏍囩郴(榛樿鍊兼槸flase)銆?	 *            <br/>
	 *            x,y 鍒诲害鍋忕Щ閲忥紙鏆傛椂鏈敮鎸侊級<br/> ghost 濡傛灉鎷栨斁鐨勫璞℃槸娴姩鐨勶紝鏄惁鎷栨斁鍑虹幇褰卞瓙 ghostSize
	 *            楝煎奖鐨勫昂瀵革紝褰撹缃簡灏哄锛屽垵濮嬩綅缃氨浠ラ紶鏍囦綅缃畾浣嶄簡锛屾敞鎰忋€?ignoreTagName 蹇界暐鐨則agName.
	 *            涓€鑸敤鏉ュ拷鐣ヤ竴浜?鎺т欢绛?渚嬪 object embed autoScroll 鏄惁鑷姩婊氬睆 cursor 榧犳爣 <br/>
	 *            ghostStyle 璁剧疆ghost灞傛鐨勬牱寮?	 *            <br/><br/>
	 * @returns {object} 杩斿洖鎷栨斁鎻忚堪瀵硅薄
	 * @example
	 * 			QZFL.dragdrop.registerDragdropHandler(this.titleElement,this.mainElement,{range:[0,0,'',''],x:50,y:160});
	 */
	registerDragdropHandler : function(handler, target, opts) {
		var _e = QZFL.event,
			_s = QZFL.dragdrop,
			_hDom = QZFL.dom.get(handler),
			_tDom = QZFL.dom.get(target),
			targetObject;

		opts = opts || {
			range : [null, null, null, null],
			ghost : 0
		};

		if (!(_hDom = _hDom || _tDom)) { //鍟ラ兘娌＄粰杩樼帺绁為┈锛?			return null;
		}

		// 鎷栨斁鐩爣瀵硅薄
		targetObject = _tDom || _hDom;

		if (!_hDom.id) {
			_hDom.id = "dragdrop_" + (++_s.count);
		}

		_hDom.style.cursor = opts.cursor || "move";

		_s.dragdropPool[_hDom.id] = {};

		_e.on(_hDom, "mousedown", _s.startDrag, [_hDom.id, targetObject, opts]);

		return _s.dragdropPool[_hDom.id];
	},

	/**
	 * 鍙栨秷娉ㄥ唽鎷栨嫿瀵硅薄
	 *
	 * @param {HTMLElement} handle 鎺ㄦ嫿鐨勫璞＄殑handler
	 */
	unRegisterDragdropHandler : function(handler) {
		var _hDom = QZFL.dom.get(handler),
			_e = QZFL.event;

		if (!_hDom) {
			return null;
		}

		_hDom.style.cursor = "";

		QZFL.dragdrop._oldSD && (_e.removeEvent(_hDom, "mousedown", QZFL.dragdrop._oldSD)); //骞叉帀寮傛鍓嶇殑鑰佹柟娉?
		_e.removeEvent(_hDom, "mousedown", QZFL.dragdrop.startDrag);
		delete QZFL.dragdrop.dragdropPool[_hDom.id];
	},



	startDrag : function(evt){
		QZFL.dragdrop.doStartDrag.apply(QZFL.dragdrop, arguments);
		QZFL.event.preventDefault(evt);
	},


	//鑰侀€昏緫鍏煎 begin
	dragTempId : 0,
	//鑰侀€昏緫鍏煎 end


	doStartDrag : function(evt, handlerId, target, opts){
		var _s = QZFL.dragdrop,
			_e = {};
		QZFL.object.extend(_e, evt);
		QZFL.imports(_s.path, function(){
				_s.startDrag.call(_s, _e, handlerId, target, opts);
			});
	}
};


//寰堥噸瑕侊紝淇濆瓨琚鐩栧墠鐨勮€佸紩鐢?QZFL.dragdrop._oldSD = QZFL.dragdrop.startDrag;


// Extend Dom
QZFL.element.extendFn(
/** @lends QZFL.ElementHandler.prototype */
{
	dragdrop : function(target, opts){
		var _arr = [];
		this.each(function(){
			_arr.push(QZFL.dragdrop.registerDragdropHandler(this, target, opts));
		});
		return _arr;
	},
	unDragdrop : function(target, opts){
		this.each(function(){
			_arr.push(QZFL.dragdrop.unRegisterDragdropHandler(this));
		});
	}
});



/////////////
//msgbox_shell.js
/////////////


/**
 * @fileoverview 淇℃伅鎻愮ず妗嗙粍浠跺澹抽儴鍒嗛€昏緫
 * @author scorpionxu
 * @version 1.0
 */



/**
 * 淇℃伅鎻愮ず妗嗙粍浠跺寘
 * @memberOf QZFL.widget
 * @namespace 淇℃伅鎻愮ず妗嗙粍浠跺寘
 */
QZFL.widget.msgbox = {
	/**
	 * 榛樿css鏂囦欢璺緞
	 * @type string
	 * @static
	 */
	cssPath : location.protocol + "//" + QZFL.config.resourceDomain + "/ac/qzfl/release/css/msgbox.css",

	/**
	 * jslib涓讳綋璺緞
	 * @type string
	 * @static
	 */
	path : location.protocol + "//" + QZFL.config.resourceDomain + "/ac/qzfl/release/widget/msgbox.js",

	/**
	 * 褰撳墠css鏂囦欢璺緞
	 * @private
	 * @type string
	 * @static
	 */
	currentCssPath : null,

	/**
	 * 鍔犺浇鏍峰紡鐨偆
	 * @private
	 * @param {string} [s=QZFL.widget.msgbox.cssPath] 鐨偆css璺緞
	 */
	_loadCss : function(s){
		var th = QZFL.widget.msgbox;
		s = s || th.cssPath;
		if(th.currentCssPath != s){
			QZFL.css.insertCSSLink(th.currentCssPath = s);
		}
	},

	/**
	 * 鏄剧ず淇℃伅
	 *
	 * @param {string} [msgHtml=""] 淇℃伅鏂囧瓧
	 * @param {number} [type=0] 淇℃伅鎻愮ず绫诲瀷 0~3 鎻愮ず, 4:鎴愬姛 5:澶辫触 6:鍔犺浇
	 * @param {number} [timeout=5000] 鎻愮ず淇℃伅瓒呮椂鍏抽棴锛?涓烘墜鍔ㄥ叧闂?	 * @param {object} [opts={}] 鍙€夊弬鏁板寘
	 * @param {number} [opts.topPosition] 鎻愮ず妗嗙殑楂樺害浣嶇疆
	 * @param {string} [opts.cssPath=QZFL.widget.msgbox.cssPath] 鑷畾涔夋牱寮忔枃浠?	 */
	show : function(msgHtml, type, timeout, opts){
		var _s = QZFL.widget.msgbox;

		if(typeof(opts) == 'number'){
			opts = { topPosition : opts };
		}

		opts = opts || {};

		_s._loadCss(opts.cssPath);

		QZFL.imports(_s.path, function(){
			_s.show(
				msgHtml,
				type,
				timeout,
				opts
			);
		});
	},

	/**
	 * 闅愯棌淇℃伅
	 *
	 * @param {number} timeout 寤惰繜鍏抽棴鐨勬椂闂?	 */
	hide : function(timeout){
		QZFL.imports(QZFL.widget.msgbox.path, function(){
			QZFL.widget.msgbox.hide(timeout);
		});
	}
};




/////////////
//dialog_shell.js
/////////////


/**
 * QZFL.dialog閫氱敤瀵硅瘽妗嗗姛鑳界殑澶栧３閮ㄥ垎
 * @fileOverview QZFL.dialog閫氱敤瀵硅瘽妗嗗姛鑳界殑澶栧３閮ㄥ垎
 * @version 1.0
 * @author scorpionxu
 */


/**
 * QZFL鐨勫脊鍑哄璇濇绯荤粺绫?鍙兘闇€瑕丵ZFL.dragdrop绫绘敮鎸? *
 * @namespace QZFL鐨勫脊鍑哄璇濇绯荤粺
 * @memberOf QZFL
 */
QZFL.dialog = {
	cssPath : location.protocol + '//' + QZFL.config.resourceDomain + "/qzone_v6/qz_dialog.css",
	currentCssPath : '',
	path : location.protocol + '//' + QZFL.config.resourceDomain + "/ac/qzfl/release/widget/dialog.js?max_age=864010",
	count : 0,
	instances : {},

	/**
	 * 鎸夐挳绫诲瀷鐨勬灇涓惧父鏁板寘<pre>
BUTTON_TYPE : {
	Disabled : -1,
	Normal : 0,
	Cancel : 1,
	Confirm : 2,
	Negative : 3
}</pre>

	 * @type object
	 *
	 */
	BUTTON_TYPE : {
		Disabled : -1,
		Normal : 0,
		Cancel : 1,
		Confirm : 2,
		Negative : 3
	},

	/**
 	* 鍒涘缓涓€涓柊鐨勫璇濇<br />
	* 鑰佹帴鍙ｆā鍨嬶細create : function(title, content, width, height, useTween, noBorder) 涓嶅缓璁娇鐢紝浣嗙洰鍓嶄粛鐒跺吋瀹?	*     
 	* @param {string} [title=''] 鏍囬
 	* @param {string|object} [content=''] 鍐呭锛屼互鏂囨湰鏍煎紡缁欙紝鏀寔HTML锛屾垨鑰厈 src : 'url'} 鐨勫舰寮忥紝鐢╥frame寮曞叆鎸囧畾椤甸潰
 	* @param {object} [opts={}] 楂樼骇鍔熻兘閫夐」
	* @param {number} [opts.width=300] 瀹藉害
	* @param {number} [opts.height=200] 楂樺害锛岃繖閲屼笉鏄暣涓猵opup dialog鐨勯珮搴︼紝鑰屾槸鍐呭鍖虹殑楂樺害
	* @param {number} [opts.top] 宸︿笂瑙掔旱鍧愭爣锛岄粯璁ゆ槸鍔ㄦ€佽绠楃殑浣嶇疆
	* @param {number} [opts.left] 宸︿笂瑙掓í鍧愭爣锛岄粯璁ゆ槸鍔ㄦ€佽绠楃殑浣嶇疆
	* @param {boolean} [opts.useTween=false] 鏄惁鐢ㄥ姩鐢伙紙鏍规嵁鎯呭喌鏀寔锛?	* @param {boolean} [opts.noBorder=false] 鏄惁鏃犺竟妗?	* @param {boolean} [opts.showMask=false] 鏄惁浣跨敤钂欐澘
	* @param {string} [opts.title=''] 鏍囬
	* @param {string} [opts.content=''] 鍐呭
	* @param {function} [opts.onLoad] 瀵硅瘽妗咲OM缁撴瀯缁樺埗瀹屾垚鏃剁殑onLoad浜嬩欢渚﹀惉<br />
妯″瀷涓猴細<pre>
function(o){ //o鏄瀵硅瘽妗嗘湰韬?QZFL.dialog.base 绫诲疄渚嬪紩鐢?}</pre>
	* @param {string} [opts.cssPath='http://qzonestyle.gtimg.cn/ac/qzfl/release/resource/css/dialog.css']
	* @param {string} [opts.statusContent=''] 榛樿涓嶅睍鐜帮紝鑻ユ湁鏂囨湰锛屽垯鍦ㄥ乏涓嬭灞曠ず锛屾墍浜х敓鐨勮楂樹笉璁＄畻鍦ㄥ唴瀹归珮搴︿腑
	* @param {object[]} [opts.buttonConfig=[]] 鎸夐挳瀹氫箟琛紝榛樿鏃犳寜閽?	* <pre>涓€涓寜閽厤缃?
{
	text: '鎸夐挳鏂囧瓧',
	tips: '鎸夐挳榧犳爣鎮仠鍚庣殑灏忛粍鏍囨彁绀烘枃瀛?,
	type: QZFL.dialog.BUTTON_TYPE.Normal, //鎸夐挳绫诲瀷:
		//鏈塏ormal, Confirm, Negative, Cancel, Disabled 浜旂閫夋嫨
		//鍏朵腑Cancel, Confirm, Negative 涓夌鍏宠仈鍒板璞℃暣浣撴椂闂村鐞嗗櫒onConfirm, onNegative, onCancel
		//涓旂偣鍑诲悗dialog灏嗗叧闂?	clickFn: function(){
			//to do 杩欓噷鏄洿鎺ユ寜閽偣鍑诲悗鐨勫鐞嗭紝鍙互鍜宱nConfirm绛夊悓鏃跺瓨鍦紝鍏堜簬鎵ц
		}
}</pre>

	* @returns {object} 杩斿洖涓€涓?QZFL.dialog.base 绫荤殑瀹炰緥
	* 
	* @example
	<pre>
var d = QZFL.dialog.create("Hello", 'world!<br/>sdfsfsdddddddddddddddddddddddddddddddddddddddddddddddf',
	{
		statusContent : "浣犲ソ锛孴ester",
		showMask : true,
		buttonConfig : [
			{
				type : QZFL.dialog.BUTTON_TYPE.Confirm,
				text : '鍝堝搱',
				tips : '鍝堝搱鍝堝搱'
			},
			{
				type : QZFL.dialog.BUTTON_TYPE.Cancel,
				text : '鍟ワ紵',
				tips : '鍙栨秷',
				clickFn : function(){
						alert("hehe");
					}
			},
			{
				type : QZFL.dialog.BUTTON_TYPE.Normal,
				text : '鏅?,
				tips : '閫?
			},
			{
				type : QZFL.dialog.BUTTON_TYPE.Negative,
				text : '鍚?,
				tips : '缁?
			},
			{
				type : QZFL.dialog.BUTTON_TYPE.Disabled,
				text : '浣犳潵鐐瑰晩锛?,
				tips : '鐐瑰緱浜嗕箞锛燂紵锛?
			}
		]
	});</pre>

	* @see QZFL.dialog.base
 	*/
	create : function(title, content, opts /*width,height,tween,noborder,top,left*/) {
		var t,
			args,
			dialog;
		
		if(t = (typeof(opts) != "number" || isNaN(parseInt(opts, 10)))){ //鍒ゆ柇鍏煎鏂拌€侀€昏緫
			opts = opts || {};
			args = [0, 0, opts.width, opts.height, opts.useTween, opts.noBorder];
		}else{
			opts = {
				'width' : opts
			};
			args = arguments;
		}

		t && (opts.width = args[2] || 300);
		opts.height = args[3] || 200;
		opts.useTween = !!args[4];
		opts.noBorder = !!args[5];
		opts.title = title || opts.title || '';
		opts.content = content || opts.content || '';

		dialog = new QZFL.dialog.shell(opts);
		dialog.init(opts);
		return dialog;
	},


	/**
 	* 鍒涘缓鏃犺竟妗嗙殑dialog瑙嗗浘 contributed by scorr
 	*
 	* @param {string} [content=''] 鍐呭
 	* @param {number} [width=300] 瀹藉害
 	* @param {number} [height=200] 楂樺害
	* @deprecated 澶氭涓€涓撅紝浠ュ悗涓嶈鐢ㄤ簡锛岀洿鎺ョ敤 create 鎺ュ彛涓殑 noBorder 鍙€夊弬鏁?	* @see QZFL.dialog.create
 	*/
	createBorderNone : function(content, width, height) {
		var opts = opts || {};
		opts.noBorder = true;
		opts.width = width || 300;
		opts.height = height || 200;
		return QZFL.dialog.create(null, content || '', opts);
	}
};

/**
 * 鐢ㄤ簬寮傛鏋勫缓鐨勮櫄鏂规硶瀵规帴鍣? * @private
 * @param {string} pFnName 鏄犲皠鐨勬柟娉曞悕
 * @param {object} objInstance 瑙﹀彂鏂规硶鐨勫璞? * @param {object[]} args 鍙傛暟鍒楄〃
 *
 */
QZFL.dialog._shellCall = function(pFnName, objInstance, args){
	var _s = QZFL.dialog;
	QZFL.imports(_s.path, (function(th){
				return function(){
						_s.base.prototype[pFnName].apply(th, args || []);
					};
			})(objInstance));
};

/**
 * 澶栧３瀵硅薄鏋勯€犲櫒锛屽彧鏄复鏃朵娇鐢紝鏈綋鍔犺浇鍚庡皢琚鐩? * @memberOf QZFL.dialog
 * @private
 * @constructor
 * @param {object} opts 鏍煎紡鍚?QZFL.dialog.create 涓殑 opts
 */
QZFL.dialog.shell = function(opts){
	var _s = QZFL.dialog,
		cssp = opts.cssPath || _s.cssPath;
	if(cssp != _s.currentCssPath){ //鎶婁富鏍峰紡鍔犺浇鍥炴潵
		QZFL.css.insertCSSLink(cssp);
		_s.currentCssPath = cssp;
	}

	this.opts = opts;
	this.id = ('qzDialog' + (++_s.count));
	_s.instances[this.id] = this;

	//鍏煎鑰佺増鏈?begin
	this.uniqueID = _s.count;
	//鍏煎鑰佺増鏈?end

	if(!_s.base){ //鎶婁富lib鍔犺浇鍥炴潵
		QZFL.imports(_s.path);
	}
};


/**
 * 鑾峰彇zIndex
 * @private
 * @deprecated 灏介噺涓嶈鐢ㄨ繖涓惂锛岀敤鏉ュ共鍟ュ憿锛熸病鎰忎箟鍢? * @returns {number} 杩斿洖z-index鍊? */
QZFL.dialog.shell.prototype.getZIndex = function() {
	return this.zIndex || (6000 + QZFL.dialog.count); //杩欓噷蹇呴』瑕佸悓姝ヨ繑鍥烇紝涓嶈兘鐢╛shellCall锛屽紓姝ユ€ф棤娉曡鍖呭
};


(function(fl){ //鎶婁竴涓釜shell鏂规硶閮芥槧灏勫嚭鏉?	for(var i = 0, len = fl.length; i < len; ++i){
		QZFL.dialog.shell.prototype[fl[i]] = (function(pName){
				return function(){
						QZFL.dialog._shellCall(pName, this, arguments);
					};
			})(fl[i]);
	}
})(['hide', 'unload', 'init', 'fillTitle', 'fillContent', 'setSize', 'show', 'hide', 'focus', 'blur', 'setReturnValue']);





/////////////
//confirm_shell.js
/////////////


/**
 * @fileoverview confirm淇℃伅鎻愮ず绫? * @version 1.$Rev: 1855 $
 * @author QzoneWebGroup, ($LastChangedBy: scorpionxu $)
 * @lastUpdate $Date: 2010-10-27 15:51:14 +0800 (鍛ㄤ笁, 27 鍗佹湀 2010) $
 * @requires QZFL.dialog
 * @requires QZFL.maskLayout
 */

/**
 * confirm淇℃伅鎻愮ず绫?寤鸿浣跨敤鏂扮殑鎺ュ彛锛屽鏋滃湪QZONE閲屽缓璁彧鐢≦ZONE.FP.confirm();鍏蜂綋鎺ュ彛璇峰弬瑙丗rontPage鎺ュ彛鏂囨。銆? *
 * @param {string} title 瀵硅瘽妗嗘爣棰? * @param {string} content 鍐呭
 * @param {number} config 鍙傛暟閰嶇疆,璇存槑{"type":"type 绫诲瀷锛岀被鍨嬪崰3浣?111 001(1) 锛?纭畾 010 (2)锛?鍚﹀畾 100(4) 锛?鍙栨秷锛岄粯璁や负001","icontype":"confirm鏀寔鐨勬彁绀虹被鍨?succ浠ｈ〃鎴愬姛,warn浠ｈ〃鎻愮ず,error浠ｈ〃閿欒,help浠ｈ〃闂彿","hastitle":false}
 * @constructor QZFL.widget.Confirm
 * @example var _c = new QZFL.widget.Confirm(title, content, {"type":3,"icontype":"succ","hastitle":false});
 * 			_c.show();
 */
//@param {string} title 瀵硅瘽妗嗘爣棰?//@param {string} content 鍐呭
//@param {number} type 绫诲瀷锛岀被鍨嬪崰3浣?111 001(1) 锛?纭畾 010 (2)锛?鍚﹀畾 100(4) 锛?鍙栨秷锛岄粯璁や负001
//QZFL.widget.Confirm = function(title, content, type, btnText) {



QZFL.widget.Confirm = function(title, content, opts){
	//鏈€鍏堝紑濮嬬殑鏄柊鑰佹帴鍙ｅ吋瀹?	if((typeof opts != 'undefined') && (typeof opts != 'object')){
		opts = {
			type : opts,
			tips : arguments[3]
		};
	}

	opts = opts || {};

	var n,
		_s = QZFL.widget.Confirm,
		cssp = opts.cssPath || _s.cssPath;


	opts.title = opts.title || title || '';
	opts.content = opts.content || content || '';

	this.opts = opts;

	// begin
	this.tips = opts.tips = (opts.tips || []);
	// end

	n = (++_s.count);
	this.id = 'qzConfirm' + n;
	_s.instances[n] = this;

	if(cssp != _s.currentCssPath){ //鎶婁富鏍峰紡鍔犺浇鍥炴潵
		QZFL.css.insertCSSLink(cssp);
		_s.currentCssPath = cssp;
	}

	if(!_s.iconMap){
		QZFL.imports(_s.path);
	}
};

QZFL.widget.Confirm.TYPE = {
	OK : 1,
	NO : 2,
	OK_NO : 3,
	CANCEL : 4,
	OK_CANCEL : 5,
	NO_CANCEL : 6,
	OK_NO_CANCEL : 7
};

QZFL.widget.Confirm.count = 0;
QZFL.widget.Confirm.instances = {};

QZFL.widget.Confirm.cssPath = location.protocol + '//' + QZFL.config.resourceDomain + "/ac/qzfl/release/resource/css/confirm_by_dialog.css";
QZFL.widget.Confirm.path = location.protocol + '//' + QZFL.config.resourceDomain + "/ac/qzfl/release/widget/confirm_base.js";


/**
 * 鐢ㄤ簬寮傛鏋勫缓鐨勮櫄鏂规硶瀵规帴鍣? * @private
 *
 */
QZFL.widget.Confirm._shellCall = function(pFnName, objInstance, args){
	var _s = QZFL.widget.Confirm;
	QZFL.imports(_s.path, (function(th){
				return function(){
						_s.prototype[pFnName].apply(th, args || []);
					};
			})(objInstance));
};



(function(fl){ //鎶婁竴涓釜shell鏂规硶閮芥槧灏勫嚭鏉?	for(var i = 0, len = fl.length; i < len; ++i){
		QZFL.widget.Confirm.prototype[fl[i]] = (function(pName){
				return function(){
						QZFL.widget.Confirm._shellCall(pName, this, arguments);
					};
			})(fl[i]);
	}
})(['hide', 'show']);





/////////////
//datacenter.js
/////////////


/**
 * @fileoverview 鏁版嵁涓績 瀛楀吀鏂瑰紡绱㈠紩鐨勬暟鎹腑蹇?//to do
 * @version 1.$Rev: 1921 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2011-01-11 18:46:01 +0800 (鍛ㄤ簩, 11 涓€鏈?2011) $
 */

/**
 * 鏁版嵁涓績
 *
 * @namespace QZFL.dataCenter
 * 
 * 瀛樺湪shareObject涓殑涓滆タ鏍规湰鍙栦笉鍒? */
(function(qdc){
	var dataPool = {};
	/**
	 * 鍐呴儴瀹炰綋
	 *
	 * @ignore
	 */
	qdc.get = qdc.load = function(key){
		return dataPool[key];
	};
	/**
	 * @鍐呴儴瀹炰綋
	 * @ignore
	 */
	qdc.del = function(key){
		dataPool[key] = null;
		delete dataPool[key];
		return true;
	};
	/**
	 * 鍐呴儴瀹炰綋
	 *
	 * @ignore
	 */
	qdc.save = function saveData(key, value){
		dataPool[key] = value;
		return true;
	};
})(QZFL.dataCenter = {});




/////////////
//mask_layout.js
/////////////


/**
 * @fileoverview QZFL瀵硅瘽妗嗙被,闇€瑕丵ZFL.dragdrop绫绘敮鎸? * @version 1.$Rev: 1917 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 */


/**
 * 寮€鍚竴涓叏灞忓箷閬洊鐨勭伆鑹茶挋鏉匡紝鐏板害鍙皟鑺傦紝鍏ㄥ眬鍞竴
 * 涔熷彲浠ュ綋寤虹珛閬僵灞傜殑鏂规硶
 *
 * @namespace 钂欐澘灞傜粍浠? * @name maskLayout
 * @memberOf QZFL
 * @function
 * @see QZFL.maskLayout.create
 * @example QZFL.maskLayout();
 */
QZFL.maskLayout = (function(){
	var masker = null,
		count = 0,

		/**
		 * 寤虹珛涓€涓伄缃╁眰
		 * @memberOf QZFL.maskLayout
		 * @name create
		 * @function
		 */
		qml = function(zi, doc, opts){
			++count;

			if(masker){
				return count;
			}

			zi = zi || 5000;
			doc = doc || document;
			opts = opts || {};

			var t = parseFloat(opts.opacity, 10);
			opts.opacity = isNaN(t) ? 0.2 : t;

			t = parseFloat(opts.top, 10);
			opts.top = isNaN(t) ? 0 : t;

			t = parseFloat(opts.left, 10);
			opts.left = isNaN(t) ? 0 : t;

			masker = QZFL.dom.createElementIn("div", doc.body, false, {
				className: "qz_mask",
				unselectable: 'on'
			});

			masker.style.cssText = 'background-color:#000;-ms-filter:"alpha(opacity=20)";#filter:alpha(opacity=' + 100 * opts.opacity + ');opacity:' + opts.opacity + '; position:fixed;_position:absolute;left:' + (opts.left) + 'px;top:' + (opts.top) + 'px;z-index:' + zi + ';width:100%;height:' + QZFL.dom[QZFL.userAgent.ie < 7 ? 'getScrollHeight' : 'getClientHeight'](doc) + 'px;';

			return count;
		};

	/**
	 * 璁剧疆灞傞€忔槑搴?	 * @memberOf QZFL.maskLayout
	 * @name setOpacity
	 * @function
	 * @param {number} ov 閫忔槑搴︽瘮渚嬶紝鏁板€煎湪 [0, 1] 鍖洪棿鍐咃紝濡?0.35
	 */
	qml.setOpacity = function(ov){
		if (masker && ov) {
			QZFL.dom.setStyle(masker, 'opacity', ov);
		}
	};

	/**
	 * 鑾峰彇褰撳墠閬僵灞傜殑DOM寮曠敤
	 * @memberOf QZFL.maskLayout
	 * @name getRef
	 * @function
	 * @returns {object} 褰撳墠閬僵灞傜殑DOM寮曠敤
	 */
	qml.getRef = function(){
		return masker;
	};

	/**
	 * 绉婚櫎閬僵灞?	 * @memberOf QZFL.maskLayout
	 * @name remove
	 * @function
	 * @param {boolean} [rmAll=false] 濡傛灉鏄痶rue鍒欑珛鍗崇Щ闄わ紝濡傛灉涓篺alse鍒欏彧鍑忓皯寮曠敤璁℃暟锛岃鏁颁负0鏃跺仛瀹為檯绉婚櫎
	 */
	qml.remove = function(rmAll){
		count = Math.max(--count, 0);

		if(!count || rmAll){
			QZFL.dom.removeElement(masker);
			masker = null;

			rmAll && (count = 0);
		}
	};

	return (qml.create = qml);
})();




/////////////
//fix_layout.js
/////////////


/**
 * @fileoverview fixed灞傛晥鏋滐紝涓昏姝ｅIE6鍋氬吋瀹? * @version 1.$Rev: 1723 $
 * @author QzoneWebGroup, ($LastChangedBy: ryanzhao $)
 * @lastUpdate $Date: 2010-04-08 19:26:57 +0800 (鍛ㄥ洓, 08 鍥涙湀 2010) $
 */

/**
 * 椤甸潰涓婁笅鍖哄煙鐨勬按鍗板眰鐢熸垚鍣? *
 * @namespace QZFL.fixLayout
 */
QZFL.fixLayout = {
	_fixLayout : null,
	_isIE6 : (QZFL.userAgent.ie && QZFL.userAgent.ie < 7),

	_layoutDiv : {},
	_layoutCount : 0,

	/**
	 * 鍒濆鍖杅ixed灞?	 *
	 * @ignore
	 */
	_init : function() {
		this._fixLayout = QZFL.dom.get("fixLayout") || QZFL.dom.createElementIn("div", document.body, false, {
			id : "fixLayout",
			style : "width:100%;"
		});
		this._isInit = true;
		// document.body.onscroll = this._onscroll;
		if (this._isIE6) {
			// // document.documentElement.onscroll = QZFL.event.bind(this,
			// this._onscroll);
			QZFL.event.addEvent(document.compatMode == "CSS1Compat" ? window : document.body, "scroll", QZFL.event.bind(this, this._onscroll));
		}
	},

	/**
	 * 鍒涘缓涓€涓嚜鍔ㄤ慨姝ｇ殑灞? 涓嶅缓璁垱寤哄お澶?	 *
	 * @param {String} html html鍐呭
	 * @param {Boolean} isBottom=false 鎵€鍦ㄤ綅缃紝鏄惁澶勪簬搴曢儴,榛樿涓哄垱寤洪《閮ㄧ殑瀹瑰櫒
	 * @param {String} layerId 娴姩灞傜殑dom id
	 * @param {bool} noFixed 涓嶄慨姝ｆ诞鍔ㄥ眰鐨勪綅缃?for ie6 only argument
	 * @param {Object} options 鍏朵粬鍙傛暟
	 * @return {Number} 杩斿洖杩欎釜灞傜殑id缂栧彿
	 * @example
	 * 			QZFL.fixLayout.create(html, true, "_returnTop_layout", false, {style : "right:0;z-index:5000" + (QZONE.userAgent.ie ? ";width:100%" : "")});
	 */
	create : function(html, isBottom, layerId, noFixed, options) {
		if (!this._isInit) { // 濡傛灉娌℃湁鍒濆鍖栵紝鍒欒嚜鍔ㄥ垵濮嬪寲
			this._init();
		}
		options = options || {};
		var tmp = {
			style : (isBottom ? "bottom:0;" : "top:0;") + (options.style || "left:0;width:100%;z-index:10000")
		}, _c;

		if (layerId) {
			tmp.id = layerId;
		}
		this._layoutCount++;
		_c = this._layoutDiv[this._layoutCount] = QZFL.dom.createElementIn("div", this._fixLayout, false, tmp);
		_c.style.position = this._isIE6 ? "absolute" : "fixed";
		_c.isTop = !isBottom;
		_c.innerHTML = html;
		_c.noFixed = noFixed ? 1 : 0;
		return this._layoutCount;
	},

	/**
	 * 鎶婂浐瀹氬眰鍥哄畾鍒伴《閮?	 *
	 * @param {number} layoutId 灞傜紪鍙?- 鐢眂reate鐨勬椂鍊欒繑鍥炵殑缂栧彿
	 */
	moveTop : function(layoutId) {
		if (!this._layoutDiv[layoutId].isTop) {
			with (this._layoutDiv[layoutId]) {
				if (this._isIE6 && !this._layoutDiv[layoutId].noFixed) {
					style.marginTop = QZFL.dom.getScrollTop() + "px";
					style.marginBottom = "0";
					style.marginBottom = "auto";
				}
				style.top = "0";
				style.bottom = "";
				isTop = true;
			}
		}
	},

	/**
	 * 鎶婂浐瀹氬眰鍥哄畾鍒板簳閮?	 *
	 * @param {number} layoutId 灞傜紪鍙?- 鐢眂reate鐨勬椂鍊欒繑鍥炵殑缂栧彿
	 */
	moveBottom : function(layoutId) {
		if (this._layoutDiv[layoutId].isTop) {
			with (this._layoutDiv[layoutId]) {
				if (this._isIE6 && !this._layoutDiv[layoutId].noFixed) {
					style.marginTop = "auto";
					style.marginBottom = "0";
					style.marginBottom = "auto";
				}
				style.top = "";
				style.bottom = "0";
				isTop = false;
			}
		}
	},

	/**
	 * 寰€鍥哄畾灞傞噷濉厖html鍐呭html
	 *
	 * @param {number} layoutId 灞傜紪鍙?- 鐢眂reate鐨勬椂鍊欒繑鍥炵殑缂栧彿
	 * @param {string} html html鍐呭
	 */
	fillHtml : function(layoutId, html) {
		this._layoutDiv[layoutId].innerHTML = html;
	},

	/**
	 * 鐩戝惉婊氬姩鏉★紝for ie6
	 *
	 * @ignore
	 */
	_onscroll : function() {
		var o = QZFL.fixLayout;
		for (var k in o._layoutDiv) {
			if (o._layoutDiv[k].noFixed) {
				continue;
			}
			QZFL.dom.setStyle(o._layoutDiv[k], "display", 'none');
		}
	
		clearTimeout(this._timer);
		this._timer = setTimeout(this._doScroll, 500);

		if (this._doHide) {
			return
		}

		this._doHide = true;
	},

	/**
	 * 鎵ц婊氬姩鏉℃粴鍔ㄥ悗鐨勮剼鏈?for ie6
	 *
	 * @ignore
	 */
	_doScroll : function() {
		var o = QZFL.fixLayout;

		for (var k in o._layoutDiv) {
			if (o._layoutDiv[k].noFixed) {
				continue;
			}
			var _item = o._layoutDiv[k];
			if (_item.isTop) {
				o._layoutDiv[k].style.marginTop = QZFL.dom.getScrollTop() + "px";
			} else {
				// ie6鐨刡ottom 鏈塨ug...
				// 蹇呴』瑕侀噸鏂拌缃竴涓嬶紝鐒跺悗鍦ㄨ繕鍘焟arginBottom鍗冲彲杩樺師銆傚穿婧冧簡,鎯虫鐨勫績閮芥湁浜?				o._layoutDiv[k].style.marginBottom = "0";
				o._layoutDiv[k].style.marginBottom = "auto";
			}
		}
		
		clearTimeout(this._stimer);
		this._stimer = setTimeout(function(){
			for (var k in o._layoutDiv) {
				if (o._layoutDiv[k].noFixed) {
					continue;
				}
				QZFL.dom.setStyle(o._layoutDiv[k], "display", "");
			}
		},800);

		o._doHide = false;
	}
};




/////////////
//tips_shell.js
/////////////


/**
 * @fileoverview 鎸囧悜鎬ф彁绀烘 澶栧３
 * @version 1.$Rev: 1821 $
 * @author QzoneWebGroup
 * @lastUpdate $Date: 2010-09-26 12:22:44 +0800 (鍛ㄦ棩, 26 涔濇湀 2010) $
 */





/**
 * @fileoverview 姘旀场鎻愮ず妗? * @version 1.$Rev: 1869 $
 * @author QzoneWebGroup
 * @lastUpdate $Date: 2010-11-03 12:37:14 +0800 (鍛ㄤ笁, 03 鍗佷竴鏈?2010) $
 */

/**
 * 姘旀场 widget
 * 
 * @namespace
 */
QZFL.widget.bubble = {
	/**
	 *
	 *path鍦ㄦ湰鏂囦欢灏捐鎸囧悜QZFL.widget.path
	 */
//	path : "http://qzonestyle.gtimg.cn/ac/qzfl/release/widget/tips.js",

	/**
	 * 姘旀场鎻愮ず鎺ュ彛
	 * 
	 * @param {Element} target 瀵硅薄
	 * @param {String} title 姘旀场鏍囬
	 * @param {String} msg 姘旀场鍐呭
	 * @param {Object} opts 姘旀场鍙傛暟 {timeout,className,id,styleText,call}
	 * @return 杩斿洖鏂板缓鐨勬皵娉＄紪鍙?	 */
	show : function(target, title, msg, opts) {
		opts = opts || {};
		var bid = opts.id || "oldBubble_" + (++QZFL.widget.bubble.count);
		opts.id = bid;
		QZFL.imports(QZFL.widget.bubble.path, function(){
			QZFL.widget.tips.show(
				'<div>' + title + '</div>' + msg,
				target,
				opts
			);
		});
		return bid;
	},

	/*
	 * 鍒濆璁℃暟
	 */
	count: 0,

	/**
	 * 闅愯棌姘旀场
	 * 
	 * @param {Number} id 姘旀场缂栧彿
	 */
	hide : function(id) {
		if(QZFL.widget.tips){
			QZFL.widget.tips.close(id);
		}
	},

	/**
	 * 闅愯棌鎵€鏈夋皵娉?	 */
	hideAll : function() {
		if(QZFL.widget.tips){
			QZFL.widget.tips.closeAll();
		}
	}
};

/**
 * 闅愯棌姘旀场
 * @deprecated
 * @param {String} bubbleId 姘旀场缂栧彿
 */
function hideBubble(bubbleId) {
	QZFL.widget.bubble.hide(bubbleId);
}

/**
 * 闅愯棌鎵€鏈夋皵娉? * @deprecated
 */
function hideAllBubble() {
	QZFL.widget.bubble.hideAll();
}
/**
 * @fileoverview 姘旀场鍔熻兘鎵╁睍
 * @version 1.$Rev: 1869 $
 * @author QzoneWebGroup
 * @lastUpdate $Date: 2010-11-03 12:37:14 +0800 (鍛ㄤ笁, 03 鍗佷竴鏈?2010) $
 */

/**
 * extend Bubble Create
 * @deprecated
 * @param {String} key 鍏抽敭key
 * @param {Element} target 瀵硅薄
 * @param {String} msg 姘旀场鏍囬
 * @param {Object} options 姘旀场鍙傛暟 {timeout,className,id,styleText,callback}
 * @return 杩斿洖鏂板缓鐨勬皵娉＄紪鍙? */
QZFL.widget.bubble.showEx = QZFL.emptyFn;

/**
 * 璁剧疆姘旀场鏄惁涓嶅啀鏄剧ず鐨刱ey
 * @deprecated
 * @param {string} key shareObject 鐨?key
 * @param {bool} value 璁剧疆涓嶅湪鏄剧ず鐨勫€? */
QZFL.widget.bubble.setExKey =  QZFL.emptyFn;


/**
 * 姘旀场 widget
 * 
 * @namespace
 */
QZFL.widget.tips = {
	/**
	 *
	 *
	 */
	path : location.protocol + '//' + QZFL.config.resourceDomain + "/ac/qzfl/release/widget/tips.js?max_age=1209600",

	/**
	 * 姘旀场灞曠ず
	 *
	 * @param {string} [html = ""] 鍐呭html
	 * @param {HTMLElement} [aim = document.body] 鎵€鎸囩洰鏍囪妭鐐?	 * @param {object} [opts = {
			 x:0, //涓庨粯璁よ绠楃殑瀵归綈浣嶇疆鐨勬í鍚戜究瀹滐紝姝ｈ礋鏁存暟鐨嗗彲
			 y:0, //涓庨粯璁よ绠楃殑瀵归綈浣嶇疆鐨勭旱鍚戜究瀹滐紝姝ｈ礋鏁存暟鐨嗗彲
			 width:200, //瀹藉害
			 height:100, //楂樺害
			 arrowType:2 //2锛?0搴﹁寮忔牱锛岀粡鍏稿満鏅?1锛?5搴﹁寮忔牱锛岄€傚悎鈥滆皝璋佽銆傘€傘€傗€濆満鏅?			 arrowSize:(鏍规嵁arrowType鍐冲畾) //灏忓皷瑙抸oom澶у皬
			 arrowEdge:(鏍规嵁aim鑺傜偣浣嶇疆鍔ㄦ€佸喅瀹? //灏栬鎵€鍦ㄧ殑杈?1锛氫笂 2锛氬彸 3锛氫笅 4锛氬乏
			 arrowPoint:(鏍规嵁aim鑺傜偣浣嶇疆鍜宎rrowEdge鍔ㄦ€佸喅瀹? //灏栬鎵€闈犺繎鐨勭鐐?1锛氬乏涓?2锛氬彸涓?3锛氬彸涓?4锛氬乏涓?			 arrowOffset:(鏍规嵁arrowtype鍜宎rrowSize鍔ㄦ€佽绠? //灏栬璺濈鎵€鎸囧畾鐨勨€滈潬杩戠鐐光€濈殑鍋忕Щ
			 borderColor:'#cbae85',
			 backgroundColor:'#fdfdd9',
			 backgroundImageUrl:"", //鏁翠釜鍐呭鍖鸿儗鏅浘URL
			 appendMode:0, //鎻掑叆椤甸潰鐨勬柟寮忥細
			 //   0 [榛樿]缁濆瀹氫綅鍦ㄩ〉闈ody涓婃彃鍏?			 //   1 鎻掑湪aim涔嬪悗锛屼笌aim骞跺垪
			 //   2 鎻掑湪aim涔嬪墠锛屼笌aim骞跺垪
			 noShadow:false,
			 closeButtonColor:'#ba6f2b',
			 closeButtonSize:'12', //榛樿12px鐨勫瓧浣撳ぇ灏忥紙杩欓噷浣跨敤unicode '脳' 瀛楃瀹炵幇鐨勶級
			 needFixed:false, //鏄惁浣跨敤position:fixed妯″紡
			 noQueue:false, //鏄惁涓嶅弬涓庢帓闃燂紝鎺掗槦鐨勬剰鎬濇槸锛氬綋鍓嶆湁涓猼ips鍦ㄥ睍绀轰簡锛岄偅涔堝氨pedding鐫€锛岀瓑褰撳墠tips鍏虫帀鍐嶆樉绀鸿嚜宸?			 timeout:5000, //鑷姩鍏抽棴鍓嶅仠鐣欐椂闂达紝闇€瑕佹亽瀹氬瓨鍦ㄥ彲浠ョ粰 -1
			 callback:undefined //鍏抽棴鍚庣殑鍥炶皟鍑芥暟
		 }]
	 * @return {string} 瀹炰綋鐨刬d
	 */
	show : function(html, aim, opts) {
		opts = opts || {};
		var bid = opts.id || "QZFL_bubbleTips_" + (++QZFL.widget.tips.count);
		opts.id = bid;
		QZFL.imports(QZFL.widget.tips.path, function(){
			QZFL.widget.tips.show(
				html,
				aim,
				opts
			);
		});
		return bid;
	},

	/*
	 * 鍒濆璁℃暟
	 */
	count: -1,

	/**
	 * 闅愯棌姘旀场
	 * 
	 * @param {Number} id 姘旀场缂栧彿
	 */
	close : function(id) {
		QZFL.imports(QZFL.widget.tips.path, function(){
			if(QZFL.widget.tips){
				QZFL.widget.tips.close(id);
			}
		});
	},

	/**
	 * 闅愯棌鎵€鏈夋皵娉?	 */
	closeAll : function() {
		QZFL.imports(QZFL.widget.tips.path, function(){
			if(QZFL.widget.tips){
				QZFL.widget.tips.closeAll();
			}
		});
	},
	resize : function(id) {
		QZFL.imports(QZFL.widget.tips.path, function(){
			if(QZFL.widget.tips){
				QZFL.widget.tips.resize && QZFL.widget.tips.resize(id);
			}
		});
	}
};
//鍔犱釜鍒ゆ柇锛屾€曞共鎺塓ZFL.widget.bubble鐨勬椂鍊欑枏蹇戒簡娌″共鎺夎繖鍙ヤ唬鐮佸氨鎶ラ敊浜?(QZFL.widget.bubble || {}).path = QZFL.widget.tips.path;



/////////////
//seed.js
/////////////


/*
 * Copyright (c) 2008, Tencent Inc. All rights reserved.
 */
/**
 * @fileoverview 闅忔満seed
 * @author QzoneWebGroup
 * @version 1.0
 */

/**
 * 鐢ㄦ潵杩斿洖
 *
 * @namespace
 */
QZFL.widget.seed = {
	_seed : 1,

	/**
	 * 璁板綍cooki鐨勫煙鍚?	 */
	domain : "qzone.qq.com",
	
	prefix : "__Q_w_s_",

	/**
	 * 鏇存柊Seed鏁板€?	 * @param {string} [k] seed鐨勫悕绉?	 * @param {object} [opt] 鐩稿叧閫夐」 useCookie, domain
	 * @return {number} 鏇存柊鍚庣殑鍊硷紝鏇存柊澶辫触涓?
	 */
	update : function(k, opt) {
		var n = 1, s, th = QZFL.widget.seed;
		if (typeof(k) == "undefined") {
			n = th._update();
		}else{
			k = th.prefix + k;
			if(opt && opt.useCookie){
				n = QZFL.cookie.get(k);
				if(n){
					QZFL.cookie.set(k, ++n, opt.domain || th.domain, null, 3000)
				}else{
					return 0;
				}
			}else{
				s = QZFL.shareObject.getValidSO();

				if (!s) {
					n = th._update();
				} else if (n = s.get(k)) {
					s.set(k, ++n);
				} else {
					return 0;
				}
			}
		}
		return n;
	},
	
	_update : function(){
		var th = QZFL.widget.seed;
		QZFL.cookie.set(
			"randomSeed",
			(th._seed = parseInt(Math.random() * 1000000, 10)),
			th.domain,
			null,
			3000);
		return th._seed;
	},

	/**
	 * 鑾峰緱Seed鏁板€?	 * @param {string} [k] seed鐨勫悕绉?	 * @param {object} [opt] 鐩稿叧閫夐」 useCookie, domain
	 * @return {number} seed鍊?	 */
	get : function(k, opt) {
		var s, n, th = QZFL.widget.seed;
		if (typeof(k) == "undefined") {
			return (th._seed = QZFL.cookie.get("randomSeed")) ? th._seed : th.update();
		}else{
			k = th.prefix + k;
			if(opt && opt.useCookie){
				return (n = QZFL.cookie.get(k)) ? n : (QZFL.cookie.set(k, n = 1, opt.domain || th.domain, null, 3000), n);
			}else{
				if(!(s = QZFL.shareObject.getValidSO())){
					return th._seed;
				}
				return (n = s.get(k)) ? n : (s.set(k, n = 1), n); /* 鍛靛懙锛屽ソ涔呬笉鐢ㄩ€楀彿琛ㄨ揪寮忎簡 */
			}
		}
	}
};



/////////////
//runbox.js
/////////////


/**
 * @fileOverview 杩欐槸涓湁瓒ｇ殑widget,鑳藉鍦ㄤ袱涓狣om涔嬫湡闂寸粯鍒朵竴涓細璺戠殑澶栨 闇€瑕?QZFL.Tween 鏀寔
 * @author zishunchen, last modified by scorpionxu
 */

/**
 * 涓や釜宸茬粡缁欏嚭鐨勮妭鐐归棿璺戜竴娆¤櫄绾挎锛屾爣璇嗚矾寰? * @memberOf QZFL.widget
 * @param {object - HTMLElement} startNode 寮€濮嬭妭鐐? * @param {object - HTMLElement} endNode 缁撴潫鑺傜偣
 * @param {object} [opts={}] 閫夐」
 * @param {number} [opts.duration=0.8] 璺戜竴娆＄殑鏃堕棿
 * @param {object} [opts.doc=document] document鑺傜偣
 *
 */
QZFL.widget.runBox = function(startNode, endNode, opts){
	var doc,
		dv,
		sp,
		ep;

	startNode = QZFL.dom.get(startNode);
	endNode = QZFL.dom.get(endNode);

	if(!QZFL.lang.isNode(startNode) || !QZFL.lang.isNode(endNode) || !QZFL.effect){
		return;
	}

	opts = opts || {};
	opts.duration = opts.duration || 0.8;
	doc = opts.doc = opts.doc || document;

	sp = QZFL.dom.getPosition(startNode);

	dv = doc.createElement("div");
	dv.style.cssText = "border:3px solid #999; z-index:10000; position:absolute; left:" + sp.left + "px; top:" + sp.top + "px; width:" + sp.width + "px; height:" + sp.height + "px;";
	doc.body.appendChild(dv);

	ep = QZFL.dom.getPosition(endNode);

	QZFL.effect.run(dv, {
		left : ep.left,
		top : ep.top,
		width : ep.width,
		height : ep.height
	},{
		duration : opts.duration * 1000,
		complete : function(){
			doc.body.removeChild(dv);
			sp = ep = dv = null;
		}
	});
};


/**
 * 鍚孮ZFL.widget.runBox
 * @deprecated 鍚嶅瓧澶暱锛岀敤QZFL.widget.runBox鍚? * @see QZFL.widget.runBox
 *
 */
QZFL.widget.runBox.start = function(){
	QZFL.widget.runBox.apply(QZFL.widget.runBox, arguments);
};



/////////////
//global_var.js
/////////////





/**
 * @author scorr
 */


/**
 * 闇€瑕佹墦寮€string鍛藉悕绌洪棿
 */
QZFL.object.map(QZFL.string || {});

/**
 * 闇€瑕佹墦寮€util鍛藉悕绌洪棿
 */
QZFL.object.map(QZFL.util || {});

/**
 * 闇€瑕佹墦寮€lang鍛藉悕绌洪棿
 */
QZFL.object.map(QZFL.lang || {});

(function(w){
	w.ua = w.ua || QZFL.userAgent;
	w.$e = QZFL.element.get;
	!w.$ && (w.$ = QZFL.dom.get);
	w.removeNode = QZFL.dom.removeElement;
	w.ENV = QZFL.enviroment;
	w.addEvent = QZFL.event.addEvent;
	w.removeEvent = QZFL.event.removeEvent;
	w.getEvent = QZFL.event.getEvent;
	w.insertFlash = QZFL.media.getFlashHtml;

	w.getShareObjectPrefix = getShareObjectPrefix; //flash so 涓鐢ㄧ殑鍥炶皟

})(window);



/////////////
//qzfl_common_anti_csrf.js
/////////////


if(!QZFL.pluginsDefine){
	QZFL.pluginsDefine = {};
}

/**
 * 鑾峰彇鍙岰SRF鐨則oken
 * @author scorpionxu
 * @example QZONE.FrontPage.getACSRFToken();
 * @return {String} 杩斿洖token涓? *
 */
QZFL.pluginsDefine.getACSRFToken = function(url){
	//鏈変紶url锛屼笖url鏄痲q.com鑰屼笉鏄痲zone.qq.com锛岃鐢╯key绠?	if(url && url.indexOf('qzone.qq.com')==-1 && url.indexOf('qq.com')>-1){
		return arguments.callee._DJB(QZFL.cookie.get("ovb_open_access_token") || QZFL.cookie.get("skey"));
	}
	//openid鏀归€?濡傛灉鏄痯_skey鎴杝key鐨勮瘽锛屽厛鍙杘vb_open_access_token
	return arguments.callee._DJB(QZFL.cookie.get("ovb_open_access_token") || QZFL.cookie.get("p_skey") || QZFL.cookie.get("skey"));
};

/**
 * 涓€涓畝鍗曠殑鎽樿绛惧悕绠楁硶
 * @author scorpionxu
 * @ignore
 */
QZFL.pluginsDefine.getACSRFToken._DJB = function(str){
	var hash = 5381;

	for(var i = 0, len = str.length; i < len; ++i){
		hash += (hash << 5) + str.charCodeAt(i);
	}

	return hash & 0x7fffffff;
};




/////////////
//qzfl_formsender_anti_csrf.js
/////////////


(function(){
	var t = QZONE.FormSender;


	if(t && t.pluginsPool){
		t.pluginsPool.formHandler.push(function(fm){
			if(fm){
				if(!fm.g_tk){
				/*	var ipt = QZFL.dom.createNamedElement("input", "g_tk", document);
					ipt.type = "hidden";
					ipt.value = QZFL.pluginsDefine.getACSRFToken();
					fm.appendChild(ipt);*/

					var a = QZFL.string.trim(fm.action);
					a += (a.indexOf("?") > -1 ? "&" : "?") + "g_tk=" + QZFL.pluginsDefine.getACSRFToken();
					fm.action = a;
				}
			}

			return fm;
		});
	}
})();



/////////////
//qzfl_jsongetter_anti_csrf.js
/////////////


/*(function(){
	var t = QZONE.JSONGetter;


	if(t && t.pluginsPool){
		t.pluginsPool.srcStringHandler.push(function(ss){
			if(typeof(ss) == "string"){
				if(ss.indexOf("g_tk=") < 0){
					ss += (ss.indexOf("?") > -1 ? "&" : "?") + "g_tk=" + QZFL.pluginsDefine.getACSRFToken();
				}
			}
			return ss;
		});
	}
})();*/

(function(){
	var t = QZONE.JSONGetter,
		jsRE = /\.js|\.json$/i;


	if(t && t.pluginsPool){
		t.pluginsPool.srcStringHandler.push(function(ss){
			var sw, pn;
			if(typeof(ss) == "string"){
				if(ss.indexOf("g_tk=") < 0){
					pn = (sw = (ss.indexOf("?") > -1)) ? ss.split('?')[0] : ss;
					if(jsRE.lastIndex = 0, !jsRE.test(pn)){
						ss += (sw ? "&" : "?") + "g_tk=" + QZFL.pluginsDefine.getACSRFToken(ss);
					}
				}
			}
			return ss;
		});
	}
})();




/////////////
//qzfl_common_network_check.js
/////////////


if(!QZFL.pluginsDefine){
	QZFL.pluginsDefine = {};
}

QZFL.pluginsDefine.networkChectLibPath = location.protocol + '//' + QZFL.config.resourceDomain + '/qzone/v6/troubleshooter/network_check_plugin_lib.js';




/////////////
//qzfl_formsender_network_check.js
/////////////


(function(){
	var t = QZONE.FormSender;


	if(t && t.pluginsPool){
		t.pluginsPool.onErrorHandler.push(function(fsObj){
			fsObj
				&& QZFL.pluginsDefine
				&& QZFL.pluginsDefine.networkChectLibPath
				&& QZFL.imports
				&& QZFL.imports(QZFL.pluginsDefine.networkChectLibPath, (function(d){
					return function(){
						QZONE
							&& QZONE.troubleShooter
							&& QZONE.troubleShooter.qzflPluginNetworlCheck
							&& QZONE.troubleShooter.qzflPluginNetworlCheck(d);
					};
				})({ url : fsObj.uri }));
		});
	}
})();



/////////////
//qzfl_jsongetter_network_check.js
/////////////


(function(){
	var t = QZONE.JSONGetter;


	if(t && t.pluginsPool){
		t.pluginsPool.onErrorHandler.push(function(jgObj){
			jgObj
				&& QZFL.pluginsDefine
				&& QZFL.pluginsDefine.networkChectLibPath
				&& QZFL.imports
				&& QZFL.imports(QZFL.pluginsDefine.networkChectLibPath, (function(d){
					return function(){
						QZONE
							&& QZONE.troubleShooter
							&& QZONE.troubleShooter.qzflPluginNetworlCheck
							&& QZONE.troubleShooter.qzflPluginNetworlCheck(d);
					};
				})({ url : jgObj._uri }));

		});
	}
})();




/////////////
//common_value_stat.js
/////////////


(function(q){
	var commurl = location.protocol==='https:'?'//huatuocode.weiyun.com/code.cgi':'//c.isdspeed.qq.com/code.cgi'
		, urlParse = /^https?:\/\/([\s\S]*?)(\/[\s\S]*?)(?:\?|$)/
		, pingSender = q.pingSender
		, collector = []
		, timer
		, isreportG = Math.random() * 2000 < 1 // jsongetter鎴愬姛鏃剁殑鎶芥牱涓婃姤鐜?		, isreportP = Math.random() * 100 < 1 // formsender鎴愬姛鏃剁殑鎶芥牱涓婃姤鐜?		, uin = typeof g_iUin == 'undefined' ? 0 : g_iUin
		, duration = 1000
		, each = q.object.each;
	// 鍏叡涓婃姤
	function valueStat(domain, cgi, type, code, time, rate, uin, exts) {
		if(Math.random() > 1 / rate) 
			return;
		var param = [];
		param.push(
			'uin=' + uin,
			'key=' + 'domain,cgi,type,code,time,rate',
			'r=' + Math.random()
		);
		// 濡傛灉鏄暟缁?		if (typeof exts.unshift == 'function') {
			var i = 0;
			while (exts.length) {
				if (param.join('&').length > 1000) {
					break;
				}
				var c = exts.shift();
				param.push([i + 1, 1].join('_') + '=' + c[0]);
				param.push([i + 1, 2].join('_') + '=' + c[1] + '?qzfl');
				param.push([i + 1, 3].join('_') + '=' + c[2]);
				param.push([i + 1, 4].join('_') + '=' + c[3]);
				param.push([i + 1, 5].join('_') + '=' + c[4]);
				param.push([i + 1, 6].join('_') + '=' + c[5]);
				i++;
			}
		}
		if (domain != '' || i > 0) {
			q.pingSender &&  q.pingSender(commurl + '?' + param.join('&'), 1000);	
		}
	}
	
	function _r() {
		if (collector.length) {
			valueStat('','','','','','',uin, collector);	
		}
		// 闂撮殧1绉掗挓杩涜涓€娆′笂鎶?		timer = setTimeout(_r, duration);
		duration *= 1.1;
	}
	function toabs(id) {
		if (!id)
			return '';
		var ret = id;
		if (id.indexOf('://') == 4 || id.indexOf('://') == 5) {
			ret = id;
		}
		else if (id.indexOf('../') === 0) {
			ret = location.protocol + '//' + location.hostname +  '/' + id.replace(/(?:\.\.\/)*/, location.pathname.split('/').slice(1, -1 * (id.split('../').length)).join('/') + '/' );
		}
		else if(/^[^\/]+\//.test(id) || id.indexOf('./') === 0) {
			if (id.indexOf('./') === 0) {
				id = id.substring(2);
			}
			ret = location.protocol + '//' + location.hostname + location.pathname.split('/').slice(0, -1).join('/') + '/' + id;
		}
		else if (id.charAt(0) === '/') {
			ret = location.protocol + '//' + location.hostname + id;
		}
		return ret;
	}
	each(['JSONGetter', 'FormSender'], function(n) {
		q[n].prototype.setReportRate = function (rate) {
			this.reportRate = rate;
		};
		if (q[n] && q[n].pluginsPool) {
			if (typeof q[n].pluginsPool.onRequestComplete == 'undefined') {
				q[n].pluginsPool.onRequestComplete = [];
			}
			q[n].pluginsPool.onRequestComplete.push(function (th) {
				var u = th._uri || th.uri;
				u = toabs(u);
				var mtch = u.match(urlParse), url = mtch[2], domain = mtch[1];
				// 鍑虹幇浜嗙綉缁滈敊璇粺涓€涓婃姤鍝?02鐨勭姸鎬佺爜
				if (th.msg && th.msg.indexOf('Connection') > -1) {
					collector.push([domain, url, 2, 502, +th.endTime - th.startTime, 1]);
					return;
				}
				var d = th.resultArgs;
				if (d && (d = d[0])) {
					// 娌℃湁鎺ュ叆鍏叡杩斿洖鐮佺郴缁熺殑鍙互鐩存帴杩斿洖
					if (typeof d.code == 'undefined') {
						return;
					}
					// 澶辫触鎵嶄笂鎶?					else if (d.code != 0) {
						collector.push([domain, url, 3, d.subcode || 1, +th.endTime - th.startTime, 1]);	
					}
					else {
						// 鎴愬姛鎸?/2000鎶芥牱涓婃姤
						if (th instanceof q.JSONGetter) {
							if (th.reportRate) {
								(th.reportRate == 1 || Math.random() < 1 / th.reportRate) && collector.push([domain, url, 1, d.subcode || 1, +th.endTime - th.startTime, th.reportRate || 2000]);	
							}
							else {
								isreportG && collector.push([domain, url, 1, d.subcode || 1, +th.endTime - th.startTime, th.reportRate || 2000]);	
							}
						}
						// 鎴愬姛鎸?/10鎶芥牱涓婃姤
						if (th instanceof q.FormSender) {
							if (th.reportRate) {
								(th.reportRate == 1 || Math.random() < 1 / th.reportRate) && collector.push([domain, url, 1, d.subcode || 1, +th.endTime - th.startTime, th.reportRate || 100]);	
							}
							else {
								isreportP && collector.push([domain, url, 1, d.subcode || 1, +th.endTime - th.startTime, th.reportRate || 100]);
							}
							
						}
					}
				}
			});
		}
	});
	_r();
})(QZFL);




/////////////
//footerfix.js
/////////////


if(typeof(define) === 'function'){
	define(function(){
		return QZFL;
	});
}

//(window.qzc = function(){




})();



/////////////
//global_eval.js
/////////////


/**
 * @fileOverview 闇€瑕佺敤eval瑕嗙洊绯荤粺鍘熺敓鎺ュ彛鐨刦ix閮芥斁鍦ㄨ繖閲? * @author scorr
 */


if(QZFL.userAgent.ie){ //涓€浜涙祻瑙堝櫒琛屼负鐭锛孖E9涓嶆敮鎸侀噸瀹氫箟document!
	eval((typeof document.documentMode == 'undefined' || document.documentMode < 9? "var document = QZFL._doc;" : "") + "var setTimeout = QZFL._setTimeout, setInterval = QZFL._setInterval");
}

