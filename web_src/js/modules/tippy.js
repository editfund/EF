import tippy, {followCursor} from 'tippy.js';
import {isDocumentFragmentOrElementNode} from '../utils/dom.js';
import {formatDatetime} from '../utils/time.js';

const visibleInstances = new Set();
const arrowSvg = `<svg width="16" height="7"><path d="m0 7 8-7 8 7Z" class="tippy-svg-arrow-outer"/><path d="m0 8 8-7 8 7Z" class="tippy-svg-arrow-inner"/></svg>`;

export function createTippy(target, opts = {}) {
  // the callback functions should be destructured from opts,
  // because we should use our own wrapper functions to handle them, do not let the user override them
  const {onHide, onShow, onDestroy, role, theme, arrow, ...other} = opts;

  const instance = tippy(target, {
    appendTo: document.body,
    animation: false,
    allowHTML: false,
    hideOnClick: false,
    interactiveBorder: 20,
    ignoreAttributes: true,
    maxWidth: 500, // increase over default 350px
    onHide: (instance) => {
      visibleInstances.delete(instance);
      return onHide?.(instance);
    },
    onDestroy: (instance) => {
      visibleInstances.delete(instance);
      return onDestroy?.(instance);
    },
    onShow: (instance) => {
      // hide other tooltip instances so only one tooltip shows at a time
      for (const visibleInstance of visibleInstances) {
        if (visibleInstance.props.role === 'tooltip') {
          visibleInstance.hide();
        }
      }
      visibleInstances.add(instance);
      return onShow?.(instance);
    },
    arrow: arrow || (theme === 'bare' ? false : arrowSvg),
    role: role || 'menu', // HTML role attribute
    theme: theme || role || 'menu', // CSS theme, either "tooltip", "menu", "box-with-header" or "bare"
    plugins: [followCursor],
    ...other,
  });

  if (role === 'menu') {
    target.setAttribute('aria-haspopup', 'true');
  }

  return instance;
}

/**
 * Attach a tooltip tippy to the given target element.
 * If the target element already has a tooltip tippy attached, the tooltip will be updated with the new content.
 * If the target element has no content, then no tooltip will be attached, and it returns null.
 *
 * Note: "tooltip" doesn't equal to "tippy". "tooltip" means a auto-popup content, it just uses tippy as the implementation.
 *
 * @param target {HTMLElement}
 * @param content {null|string}
 * @returns {null|tippy}
 */
function attachTooltip(target, content = null) {
  switchTitleToTooltip(target);

  content = content ?? target.getAttribute('data-tooltip-content');
  if (!content) return null;

  // when element has a clipboard target, we update the tooltip after copy
  // in which case it is undesirable to automatically hide it on click as
  // it would momentarily flash the tooltip out and in.
  const hasClipboardTarget = target.hasAttribute('data-clipboard-target');
  const hideOnClick = !hasClipboardTarget;

  const props = {
    content,
    delay: 100,
    role: 'tooltip',
    theme: 'tooltip',
    hideOnClick,
    placement: target.getAttribute('data-tooltip-placement') || 'top-start',
    followCursor: target.getAttribute('data-tooltip-follow-cursor') || false,
    ...(target.getAttribute('data-tooltip-interactive') === 'true' ? {interactive: true, aria: {content: 'describedby', expanded: false}} : {}),
  };

  if (!target._tippy) {
    createTippy(target, props);
  } else {
    target._tippy.setProps(props);
  }
  return target._tippy;
}

function switchTitleToTooltip(target) {
  let title = target.getAttribute('title');
  if (title) {
    // apply custom formatting to relative-time's tooltips
    if (target.tagName.toLowerCase() === 'relative-time') {
      const datetime = target.getAttribute('datetime');
      if (datetime) {
        title = formatDatetime(new Date(datetime));
      }
    }
    target.setAttribute('data-tooltip-content', title);
    // only replace if not explicitly set
    if (target.getAttribute('aria-label') !== null) {
      target.setAttribute('aria-label', title);
    }
    // keep the attribute, in case there are some other "[title]" selectors
    // and to prevent infinite loop with <relative-time> which will re-add
    // title if it is absent
    target.setAttribute('title', '');
  }
}

/**
 * Creating tooltip tippy instance is expensive, so we only create it when the user hovers over the element
 * According to https://www.w3.org/TR/DOM-Level-3-Events/#events-mouseevent-event-order , mouseover event is fired before mouseenter event
 * Some browsers like PaleMoon don't support "addEventListener('mouseenter', capture)"
 * The tippy by default uses "mouseenter" event to show, so we use "mouseover" event to switch to tippy
 * @param e {Event}
 */
function lazyTooltipOnMouseHover(e) {
  e.target.removeEventListener('mouseover', lazyTooltipOnMouseHover, true);
  attachTooltip(this);
}

// Activate the tooltip for current element.
// If the element has no aria-label, use the tooltip content as aria-label.
function attachLazyTooltip(el) {
  el.addEventListener('mouseover', lazyTooltipOnMouseHover, {capture: true});

  // meanwhile, if the element has no aria-label, use the tooltip content as aria-label
  if (!el.hasAttribute('aria-label')) {
    const content = el.getAttribute('data-tooltip-content');
    if (content) {
      el.setAttribute('aria-label', content);
    }
  }
}

// Activate the tooltip for all children elements.
function attachChildrenLazyTooltip(target) {
  for (const el of target.querySelectorAll('[data-tooltip-content]')) {
    attachLazyTooltip(el);
  }
}

export function initGlobalTooltips() {
  // use MutationObserver to detect new "data-tooltip-content" elements added to the DOM, or attributes changed
  const observerConnect = (observer) => observer.observe(document, {
    subtree: true,
    childList: true,
    attributeFilter: ['data-tooltip-content', 'title'],
  });
  const observer = new MutationObserver((mutationList, observer) => {
    const pending = observer.takeRecords();
    observer.disconnect();
    for (const mutation of [...mutationList, ...pending]) {
      if (mutation.type === 'childList') {
        // mainly for Vue components and AJAX rendered elements
        for (const el of mutation.addedNodes) {
          if (!isDocumentFragmentOrElementNode(el)) continue;
          attachChildrenLazyTooltip(el);
          if (el.hasAttribute('data-tooltip-content')) {
            attachLazyTooltip(el);
          }
        }
      } else if (mutation.type === 'attributes') {
        attachTooltip(mutation.target);
      }
    }
    observerConnect(observer);
  });
  observerConnect(observer);

  attachChildrenLazyTooltip(document.documentElement);
}

export function showTemporaryTooltip(target, content) {
  // if the target is inside a dropdown, don't show the tooltip because when the dropdown
  // closes, the tippy would be pushed unsightly to the top-left of the screen like seen
  // on the issue comment menu.
  if (target.closest('.ui.dropdown > .menu')) return;

  const tippy = target._tippy ?? attachTooltip(target, content);
  tippy.setContent(content);
  if (!tippy.state.isShown) tippy.show();
  tippy.setProps({
    onHidden: (tippy) => {
      // reset the default tooltip content, if no default, then this temporary tooltip could be destroyed
      if (!attachTooltip(target)) {
        tippy.destroy();
      }
    },
  });
}
