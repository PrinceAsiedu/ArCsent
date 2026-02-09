#!/usr/bin/env bash

_arcsent_ctl_completion() {
  local cur prev
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "ctl -config -reload -pid" -- "$cur") )
    return 0
  fi

  if [[ ${COMP_WORDS[1]} == "ctl" ]]; then
    if [[ ${COMP_CWORD} -eq 2 ]]; then
      COMPREPLY=( $(compgen -W "status health scanners findings baselines results trigger signatures export metrics" -- "$cur") )
      return 0
    fi
    case "${COMP_WORDS[2]}" in
      results)
        COMPREPLY=( $(compgen -W "latest history" -- "$cur") )
        return 0
        ;;
      signatures)
        COMPREPLY=( $(compgen -W "status update" -- "$cur") )
        return 0
        ;;
      export)
        COMPREPLY=( $(compgen -W "results baselines" -- "$cur") )
        return 0
        ;;
      trigger)
        return 0
        ;;
    esac
    if [[ "$prev" == "-format" ]]; then
      COMPREPLY=( $(compgen -W "json csv" -- "$cur") )
      return 0
    fi
  fi
}

complete -F _arcsent_ctl_completion arcsent
