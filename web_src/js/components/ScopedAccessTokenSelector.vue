<script>
import {createApp} from 'vue';
import {hideElem, showElem} from '../utils/dom.js';

const sfc = {
  props: {
    isAdmin: {
      type: Boolean,
      required: true,
    },
    noAccessLabel: {
      type: String,
      required: true,
    },
    readLabel: {
      type: String,
      required: true,
    },
    writeLabel: {
      type: String,
      required: true,
    },
  },

  computed: {
    categories() {
      const categories = [
        'activitypub',
      ];
      if (this.isAdmin) {
        categories.push('admin');
      }
      categories.push(
        'issue',
        'misc',
        'notification',
        'organization',
        'package',
        'repository',
        'user');
      return categories;
    },
  },

  mounted() {
    document.getElementById('scoped-access-submit').addEventListener('click', this.onClickSubmit);
  },

  unmounted() {
    document.getElementById('scoped-access-submit').removeEventListener('click', this.onClickSubmit);
  },

  methods: {
    onClickSubmit(e) {
      e.preventDefault();

      const warningEl = document.getElementById('scoped-access-warning');
      // check that at least one scope has been selected
      for (const el of document.getElementsByClassName('access-token-select')) {
        if (el.value) {
          // Hide the error if it was visible from previous attempt.
          hideElem(warningEl);
          // Submit the form.
          document.getElementById('scoped-access-form').submit();
          // Don't show the warning.
          return;
        }
      }
      // no scopes selected, show validation error
      showElem(warningEl);
    },
  },
};

export default sfc;

/**
 * Initialize category toggle sections
 */
export function initScopedAccessTokenCategories() {
  for (const el of document.getElementsByClassName('scoped-access-token')) {
    createApp(sfc, {
      isAdmin: el.getAttribute('data-is-admin') === 'true',
      noAccessLabel: el.getAttribute('data-no-access-label'),
      readLabel: el.getAttribute('data-read-label'),
      writeLabel: el.getAttribute('data-write-label'),
    }).mount(el);
  }
}

</script>
<template>
  <div v-for="category in categories" :key="category" class="field tw-pl-1 tw-pb-1 access-token-category">
    <label class="category-label" :for="'access-token-scope-' + category">
      {{ category }}
    </label>
    <div class="gitea-select">
      <select
        class="ui selection access-token-select"
        name="scope"
        :id="'access-token-scope-' + category"
      >
        <option value="">
          {{ noAccessLabel }}
        </option>
        <option :value="'read:' + category">
          {{ readLabel }}
        </option>
        <option :value="'write:' + category">
          {{ writeLabel }}
        </option>
      </select>
    </div>
  </div>
</template>
