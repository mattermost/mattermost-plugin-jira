// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export default class Validator {
    constructor() {
        // Our list of components we have to validate before allowing a submit action.
        this.components = new Map();
        this.anonymousComponents = [];
    }

    addComponent = (key, validateField) => {
        if (key) {
            this.components.set(key, validateField);
        } else {
            this.anonymousComponents.push(validateField);
        }
    };

    removeComponent = (key, validateField) => {
        if (key) {
            this.components.delete(key);
        } else {
            const index = this.anonymousComponents.indexOf(validateField);
            if (index !== -1) {
                this.anonymousComponents.splice(index, 1);
            }
        }
    };

    validate = () => {
        return Array.from(this.components.values()).reduce((accum, validateField) => {
            return validateField() && accum;
        }, true);
    };
}
