// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {
    useCallback,
    useEffect,
    useRef,
    useState,
} from 'react';

import {RHSTab, rhsTabIdentity} from 'types/rhs';
import {rhsTabsEqual} from 'utils/rhs_resolve';

export const RHS_TAB_PANEL_ID = 'jira-rhs-panel';

export function rhsTabDomId(tab: RHSTab): string {
    return 'jira-rhs-tab-' + rhsTabIdentity(tab).replace(':', '-');
}

export type Props = {
    tabs: RHSTab[];
    selectedTab: RHSTab;
    onSelect: (tab: RHSTab) => void;
};

type OverflowState = {
    left: boolean;
    right: boolean;
};

export default function RHSTabStrip(props: Props): JSX.Element {
    const stripRef = useRef<HTMLDivElement>(null);
    const buttonRefs = useRef<Map<string, HTMLButtonElement>>(new Map());
    const [overflow, setOverflow] = useState<OverflowState>({left: false, right: false});

    const updateOverflow = useCallback(() => {
        const el = stripRef.current;
        if (!el) {
            setOverflow({left: false, right: false});
            return;
        }
        const maxScroll = el.scrollWidth - el.clientWidth;
        setOverflow({
            left: el.scrollLeft > 1,
            right: maxScroll - el.scrollLeft > 1,
        });
    }, []);

    useEffect(() => {
        updateOverflow();
        const el = stripRef.current;
        const onScroll = () => {
            updateOverflow();
        };
        if (el) {
            el.addEventListener('scroll', onScroll);
        }
        let observer: ResizeObserver | null = null;
        if (el && typeof ResizeObserver !== 'undefined') {
            observer = new ResizeObserver(updateOverflow);
            observer.observe(el);
        }
        return () => {
            if (el) {
                el.removeEventListener('scroll', onScroll);
            }
            if (observer) {
                observer.disconnect();
            }
        };
    }, [props.tabs, updateOverflow]);

    const onTabKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>, index: number) => {
        let next = index;
        if (event.key === 'ArrowRight') {
            next = Math.min(index + 1, props.tabs.length - 1);
        } else if (event.key === 'ArrowLeft') {
            next = Math.max(index - 1, 0);
        } else if (event.key === 'Home') {
            next = 0;
        } else if (event.key === 'End') {
            next = props.tabs.length - 1;
        } else {
            return;
        }
        event.preventDefault();
        const tab = props.tabs[next];
        if (!tab || next === index) {
            return;
        }
        props.onSelect(tab);
        requestAnimationFrame(() => {
            buttonRefs.current.get(rhsTabDomId(tab))?.focus();
        });
    };

    const wrapClass = [
        'jira-rhs-tab-strip-wrap',
        overflow.left ? 'jira-rhs-tab-strip-wrap--fade-left' : '',
        overflow.right ? 'jira-rhs-tab-strip-wrap--fade-right' : '',
    ].filter(Boolean).join(' ');

    return (
        <div
            className={wrapClass}
            data-testid='rhs-tab-strip'
        >
            <div
                className='jira-rhs-tab-strip'
                role='tablist'
                aria-orientation='horizontal'
                ref={stripRef}
            >
                {props.tabs.map((item, index) => {
                    const selected = rhsTabsEqual(item, props.selectedTab);
                    const tabId = rhsTabDomId(item);
                    return (
                        <button
                            key={rhsTabIdentity(item)}
                            id={tabId}
                            className={selected ? 'jira-rhs-tab jira-rhs-tab--selected' : 'jira-rhs-tab'}
                            role='tab'
                            aria-selected={selected}
                            aria-controls={RHS_TAB_PANEL_ID}
                            tabIndex={selected ? 0 : -1}
                            type='button'
                            ref={(node) => {
                                if (node) {
                                    buttonRefs.current.set(tabId, node);
                                } else {
                                    buttonRefs.current.delete(tabId);
                                }
                            }}
                            onClick={() => props.onSelect(item)}
                            onKeyDown={(event) => onTabKeyDown(event, index)}
                        >
                            {item.name}
                        </button>
                    );
                })}
            </div>
        </div>
    );
}
